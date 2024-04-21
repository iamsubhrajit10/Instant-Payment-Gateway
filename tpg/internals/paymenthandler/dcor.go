package paymenthandler

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
	"tpg/config"
	pb "tpg/protos"
	resolverpb "tpg/resolverproto"

	"github.com/labstack/echo/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	GrpcClientMap        = make(map[string]pb.DetailsClient)
	GrpcConnectionMap    = make(map[string]*grpc.ClientConn)
	GrpcConnectionMapRes = make(map[string]*grpc.ClientConn)
	GrpcClientMapRes     = make(map[string]resolverpb.DetailsClient)
)

type RequestDataBank struct {
	TransactionID string
	AccountNumber string
	IFSCCode      string
	HolderName    string
	Amount        int
	Type          string
}

type Apidata struct {
	PayerPaymentID string `json:"PayerPaymentID"`
	PayeePaymentID string `json:"PayeePaymentID"`
	Amount         int    `json:"Amount"`
}
type RequestDataResolver struct {
	TransactionID string
	PaymentID     string
	Type          string
}
type ReplyDataResolver struct {
	TransactionID string `json:"TransactionID"`
	PaymentID     string `json:"PaymentID"`
	Status        string `json:"Status"`
	AccountNumber string `json:"AccountNumber"`
	IFSCCode      string `json:"IFSCCode"`
	HolderName    string `json:"HolderName"`
}

var RequestCount = 0

func getGRPCConnection(address string) (*grpc.ClientConn, error) {
	config.Logger.Print("grpc address: ", address)
	//addr := flag.String("addr", address, "the address to connect to")
	//config.Logger.Print("grpc address1: ", addr)
	if _, ok := GrpcConnectionMap[address]; !ok {
		conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			config.Logger.Fatalf("did not connect: %v", err)
			return nil, err
		}
		GrpcConnectionMap[address] = conn
	}
	return GrpcConnectionMap[address], nil
}

func getGRPCConnectionResolver(address string) (*grpc.ClientConn, error) {
	// fmt.Println(address)
	if _, ok := GrpcConnectionMapRes[address]; !ok {
		conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			config.Logger.Fatalf("did not connect: %v", err)
			return nil, err
		}
		GrpcConnectionMapRes[address] = conn
	}
	return GrpcConnectionMapRes[address], nil
}

func getGRPCClient(address string) (pb.DetailsClient, error) {
	if _, ok := GrpcClientMap[address]; !ok {
		ClientConn, err := getGRPCConnection(address)
		if err != nil {
			config.Logger.Fatalf("did not connect: %v", err)
			return nil, err
		}
		c := pb.NewDetailsClient(ClientConn)
		GrpcClientMap[address] = c
	}
	return GrpcClientMap[address], nil
}

func getGRPCClientResolver(address string) (resolverpb.DetailsClient, error) {
	if _, ok := GrpcClientMapRes[address]; !ok {
		ClientConn, err := getGRPCConnectionResolver(address)
		if err != nil {
			config.Logger.Fatalf("did not connect: %v", err)
			return nil, err
		}
		c := resolverpb.NewDetailsClient(ClientConn)
		GrpcClientMapRes[address] = c
	}
	return GrpcClientMapRes[address], nil
}

func ReverseDebit(bankServerIPV4 string, data RequestDataBank) (string, error) {

	client, err := getGRPCClient(bankServerIPV4)
	if err != nil {
		return "GRPC client not found, debit failed", err
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	//defer cancel()
	jsonString, err := json.Marshal(data)

	res, err := client.UnarryCall(ctx, &pb.Clientmsg{Name: string(jsonString)})
	if err != nil {
		config.Logger.Fatalf("could not greet: %v", err)
		return "Debit Reverse failed", err
	}
	config.Logger.Printf("success debit reverse: %s", res.GetMessage())
	return res.GetMessage(), nil
}

func DebitRequest(bankServerIPV4 string, data RequestDataBank) (string, error) {

	//defer ClientConn.Close()
	client, err := getGRPCClient(bankServerIPV4)
	if err != nil {
		return "GRPC client not found, debit failed", err
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	//defer cancel()
	jsonString, err := json.Marshal(data)

	res, err := client.UnarryCall(ctx, &pb.Clientmsg{Name: string(jsonString)})
	if err != nil {
		config.Logger.Fatalf("could not greet: %v", err)
		return "Debit Request failed", err
	}
	config.Logger.Printf("success debit: %s", res.GetMessage())
	return res.GetMessage(), nil
}

func resolveRequest(bankServerIPV4 string, data RequestDataResolver) (string, error) {
	client, err := getGRPCClientResolver(bankServerIPV4)
	if err != nil {
		return "GRPC client not found, resolve failed", err
	}
	// defer ClientConn.Close()
	// c := pb.NewDetailsClient(ClientConn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	jsonString, err := json.Marshal(data)

	r, err := client.UnarryCall(ctx, &resolverpb.Clientmsg{Name: string(jsonString)})
	if err != nil {
		config.Logger.Fatalf("could not greet: %v", err)
		return "", err
	}
	config.Logger.Printf("Greeting: %s", r.GetMessage())
	return r.GetMessage(), nil
}

func CreditRequest(bankServerIPV4 string, data RequestDataBank) (string, error) {
	client, err := getGRPCClient(bankServerIPV4)
	if err != nil {
		return "GRPC client not found, credit failed", err
	}
	// defer ClientConn.Close()
	// c := pb.NewDetailsClient(ClientConn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	jsonString, err := json.Marshal(data)

	r, err := client.UnarryCall(ctx, &pb.Clientmsg{Name: string(jsonString)})
	if err != nil {
		config.Logger.Fatalf("could not greet: %v", err)
		return "", err
	}
	config.Logger.Printf("Greeting: %s", r.GetMessage())
	return "Success", nil
}

func debitRetry(addr string, data RequestDataBank) (string, error) {
	for i := 0; i < config.DebitRetries; i++ {
		msg, err := DebitRequest(addr, data)
		if err == nil && msg == "Debit request processed" {
			return "Success", nil
		}
	}
	return "Failed", nil
}

func creditRetry(addr string, data RequestDataBank) (string, error) {
	for i := 0; i < config.DebitRetries; i++ {
		_, err := CreditRequest(addr, data)
		if err == nil {
			return "Success", nil
		}
	}
	return "Failed", nil
}

func dumpTranscation(tid string, payerAddress string, payeeAddress string, Type string, amount int, payeeAno string, payeeName string, payeeIfsc string, payerAno string, payerName string, payerIfsc string) {
	data := []string{tid, payerAddress, payeeAddress, Type, strconv.Itoa(amount), payerAno, payerIfsc, payerName, payeeAno, payeeIfsc, payeeName}

	//_, err := os.Stat("Failed_Transaction.csv")
	// if os.IsNotExist((err)) {
	// 	log.Print("hello1")
	// 	_, err := os.Create("Failed_Transaction.csv")
	// 	if err != nil {
	// 		config.Logger.Fatal(err)
	// 	}
	// }
	file, err := os.OpenFile("Failed_Transaction.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		config.Logger.Fatal(err)
	}
	defer file.Close()
	//file, err := os.Open("Failed_Transaction.csv")
	//defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write(data); err != nil {
		config.Logger.Fatal(err)
	}
}

func generateTransactionID() string {

	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(100000000))
}

func TransferHandler(c echo.Context) error {
	// Get the resolver server address

	RequestCount++
	DebitPort := config.DebitBankServerPort + RequestCount%3
	CreditPort := config.CreditBankServerPort + RequestCount%3

	resolverServerIPV4 := config.ResolverServerIPV4 + ":" + config.ResolverServerPort
	// Get the bank server address
	debitBankServerIPV4 := config.DebitBankServerIPV4 + ":" + strconv.Itoa(DebitPort)
	creitBankServerIPV4 := config.CreditBankServerIPV4 + ":" + strconv.Itoa(CreditPort)

	//dumpTranscation("1", debitBankServerIPV4, creitBankServerIPV4, "credit", 100)
	config.Logger.Println("Debit Bank Server IPV4: ", debitBankServerIPV4)
	config.Logger.Println("Credit Bank Server IPV4: ", creitBankServerIPV4)
	//scheduler.Reverse()
	// Create the request data for payer
	u := Apidata{}
	if err := c.Bind(&u); err != nil {
		return c.String(http.StatusBadRequest, "Invalid request")
	}

	config.Logger.Println("Api data: ", u)
	tid := generateTransactionID()
	resolveDataPayer := RequestDataResolver{
		TransactionID: tid,
		PaymentID:     u.PayerPaymentID,
		Type:          "resolve",
	}

	// Create the request data for payee
	resolveDataPayee := RequestDataResolver{
		TransactionID: tid,
		PaymentID:     u.PayeePaymentID,
		Type:          "resolve",
	}

	// Create channels to receive the results
	payerChan := make(chan ReplyDataResolver)
	payeeChan := make(chan ReplyDataResolver)
	errChan := make(chan error)

	// Resolve Payer PaymentID in a goroutine
	go func() {
		resolverResponsePayer, err := resolveRequest(resolverServerIPV4, resolveDataPayer)
		if err != nil {
			errChan <- fmt.Errorf("Payer Resolve Failed: %w", err)
			return
		}
		config.Logger.Printf("Payer Response: %s", resolverResponsePayer)

		var replyResolverPayer ReplyDataResolver
		err = json.Unmarshal([]byte(resolverResponsePayer), &replyResolverPayer)
		if err != nil {
			errChan <- fmt.Errorf("Failed to unmarshal response from resolver for payer: %w", err)
			return
		}
		payerChan <- replyResolverPayer
	}()

	// Resolve Payee PaymentID in a goroutine
	go func() {
		resolveResponsePayee, err := resolveRequest(resolverServerIPV4, resolveDataPayee)
		if err != nil {
			errChan <- fmt.Errorf("Payee Resolve Failed: %w", err)
			return
		}
		config.Logger.Printf("Payee Response: %s", resolveResponsePayee)

		var replyResolverPayee ReplyDataResolver
		err = json.Unmarshal([]byte(resolveResponsePayee), &replyResolverPayee)
		if err != nil {
			errChan <- fmt.Errorf("Failed to unmarshal response from resolver for payee: %w", err)
			return
		}
		payeeChan <- replyResolverPayee
	}()

	// Wait for both goroutines to finish
	replyResolverPayer, replyResolverPayee := <-payerChan, <-payeeChan

	// Check for errors
	select {
	case err := <-errChan:
		return c.String(http.StatusInternalServerError, err.Error())
	default:
		// Continue with your code
	}

	// check if the account number is not found for any of the parties
	if replyResolverPayer.Status == "not found" {
		return c.String(http.StatusBadRequest, "Payer Account not found")
	}
	if replyResolverPayee.Status == "not found" {
		return c.String(http.StatusBadRequest, "Payee Account not found")
	}

	// check if the account number is found for both
	if replyResolverPayer.Status == "found" && replyResolverPayee.Status == "found" {
		config.Logger.Printf("Payer: %s", replyResolverPayer.AccountNumber)
		config.Logger.Printf("Payee: %s", replyResolverPayee.AccountNumber)
		debitData := RequestDataBank{
			TransactionID: replyResolverPayer.TransactionID,
			AccountNumber: replyResolverPayer.AccountNumber,
			IFSCCode:      replyResolverPayer.IFSCCode,
			HolderName:    replyResolverPayer.HolderName,
			Amount:        u.Amount,
			Type:          "debit",
		}
		creditData := RequestDataBank{
			TransactionID: replyResolverPayee.TransactionID,
			AccountNumber: replyResolverPayee.AccountNumber,
			IFSCCode:      replyResolverPayee.IFSCCode,
			HolderName:    replyResolverPayee.HolderName,
			Amount:        u.Amount,
			Type:          "credit",
		}
		// Debit the payer and credit the payee
		x, err := DebitRequest(debitBankServerIPV4, debitData)

		if x != "Debit request processed" {
			return c.String(http.StatusInternalServerError, x)
		}
		//dumpTranscation(replyResolverPayee.TransactionID, debitBankServerIPV4, creitBankServerIPV4, "debit", debitData.Amount, replyResolverPayee.AccountNumber, replyResolverPayee.HolderName, replyResolverPayee.IFSCCode, replyResolverPayer.AccountNumber, replyResolverPayer.HolderName, replyResolverPayer.IFSCCode)
		if err != nil {
			msg, _ := debitRetry(debitBankServerIPV4, debitData)
			if msg == "Failed" {

				dumpTranscation(replyResolverPayee.TransactionID, debitBankServerIPV4, creitBankServerIPV4, "debit", debitData.Amount, replyResolverPayee.AccountNumber, replyResolverPayee.HolderName, replyResolverPayee.IFSCCode, replyResolverPayer.AccountNumber, replyResolverPayer.HolderName, replyResolverPayer.IFSCCode)
				return c.String(http.StatusInternalServerError, "Debit Failed")
			}
			_, err_ := CreditRequest(creitBankServerIPV4, creditData)
			if err_ != nil {

				msg, _ := creditRetry(creitBankServerIPV4, creditData)
				if msg == "Failed" {
					dumpTranscation(replyResolverPayee.TransactionID, debitBankServerIPV4, creitBankServerIPV4, "credit", creditData.Amount, replyResolverPayee.AccountNumber, replyResolverPayee.HolderName, replyResolverPayee.IFSCCode, replyResolverPayer.AccountNumber, replyResolverPayer.HolderName, replyResolverPayer.IFSCCode)
					return c.String(http.StatusInternalServerError, "Credit Failed")
				}
				//return c.String(http.StatusInternalServerError, "Credit Failed")
			}
			return c.String(http.StatusOK, "Transfer Successful")
		}
		config.Logger.Printf("Debit: %s", x)
		_, err_ := CreditRequest(creitBankServerIPV4, creditData)
		if err_ != nil {

			msg, _ := creditRetry(creitBankServerIPV4, creditData)
			if msg == "Failed" {
				dumpTranscation(replyResolverPayee.TransactionID, debitBankServerIPV4, creitBankServerIPV4, "credit", creditData.Amount, replyResolverPayee.AccountNumber, replyResolverPayee.HolderName, replyResolverPayee.IFSCCode, replyResolverPayer.AccountNumber, replyResolverPayer.HolderName, replyResolverPayer.IFSCCode)
				return c.String(http.StatusInternalServerError, "Credit Failed")
			}
			//return c.String(http.StatusInternalServerError, "Credit Failed")
		}

		return c.String(http.StatusOK, "Transfer Successful")
	}
	return c.String(http.StatusInternalServerError, "Transfer Failed")
}
