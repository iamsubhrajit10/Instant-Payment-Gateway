package paymenthandler

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
	"tpg/config"
	pb "tpg/protos"
	resolverpb "tpg/resolverproto"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	GrpcClientMap                 = make(map[string]pb.DetailsClient)
	IsThereResGrpcConnection      = make(map[string]bool)
	IsThereBankGrpcConnection     = make(map[string]bool)
	GrpcConnectionPoolBank        = make(map[string][]*grpc.ClientConn)
	GrpcConnectionPoolRes         = make(map[string][]*grpc.ClientConn)
	GrpcClientMapRes              = make(map[string]resolverpb.DetailsClient)
	successfulRequests        int = 0
	debitLimiter                  = rate.NewLimiter(2000, 4000) // 1000 requests per second, with a burst limit of 2000 requests
	creditLimiter                 = rate.NewLimiter(2000, 4000) // 1000 requests per second, with a burst limit of 2000 requests
	resolveLimiter                = rate.NewLimiter(2000, 4000) // 1000 requests per second, with a burst limit of 2000 requests
	resolverConnectionCount   int = 0
	bankConnectionCount       int = 0
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
	Requests []struct {
		TransactionID string
		PaymentID     string
		Type          string
	}
}
type ReplyResolver struct {
	TransactionID string `json:"TransactionID"`
	PaymentID     string `json:"PaymentID"`
	Status        string `json:"Status"`
	AccountNumber string `json:"AccountNumber"`
	IFSCCode      string `json:"IFSCCode"`
	HolderName    string `json:"HolderName"`
}
type ReplyDataResolver struct {
	Responses []ReplyResolver
}

func makeGRPCConnectionPoolBank(address string) error {
	var connPool []*grpc.ClientConn
	for i := 0; i < 5; i++ {
		conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return err
		}
		connPool = append(connPool, conn)
	}
	GrpcConnectionPoolBank[address] = connPool
	IsThereBankGrpcConnection[address] = true
	return nil
}
func getBankGrpcConnectionFromPool(address string) (*grpc.ClientConn, error) {
	choice := bankConnectionCount % len(GrpcConnectionPoolBank[address])
	bankConnectionCount++
	return GrpcConnectionPoolBank[address][choice], nil
}

// Round Robin Connection Pool Selection	for Bank
func getGRPCConnection(address string) (*grpc.ClientConn, error) {
	if _, ok := IsThereBankGrpcConnection[address]; !ok {
		err := makeGRPCConnectionPoolBank(address)
		if err != nil {
			return nil, err
		}
	}
	connection, _ := getBankGrpcConnectionFromPool(address)
	return connection, nil
}

// Round Robin Connection Pool Selection	for Resolver
func getResGrpcConnectionFromPool(address string) (*grpc.ClientConn, error) {
	choice := resolverConnectionCount % len(GrpcConnectionPoolRes[address])
	resolverConnectionCount++
	return GrpcConnectionPoolRes[address][choice], nil
}

func makeResGRPCConnectionPool(address string) error {
	// fmt.Println(address)
	var connPool []*grpc.ClientConn
	for i := 0; i < 5; i++ {
		conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return err
		}
		connPool = append(connPool, conn)
	}
	GrpcConnectionPoolRes[address] = connPool
	IsThereResGrpcConnection[address] = true
	return nil
}

func getGRPCConnectionResolver(address string) (*grpc.ClientConn, error) {
	// fmt.Println(address)
	if _, ok := IsThereResGrpcConnection[address]; !ok {
		err := makeResGRPCConnectionPool(address)
		if err != nil {
			return nil, err
		}
	}
	connection, _ := getResGrpcConnectionFromPool(address)
	return connection, nil
}

func getGRPCClient(address string) (pb.DetailsClient, error) {
	if _, ok := GrpcClientMap[address]; !ok {
		ClientConn, err := getGRPCConnection(address)
		if err != nil {
			log.Fatalf("did not connect: %v", err)
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
			log.Fatalf("did not connect: %v", err)
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
		log.Fatalf("could not greet: %v", err)
		return "Debit Reverse failed", err
	}
	log.Printf("success debit reverse: %s", res.GetMessage())
	return res.GetMessage(), nil
}

func DebitRequest(bankServerIPV4 string, data RequestDataBank) (string, error) {

	//defer ClientConn.Close()
	client, err := getGRPCClient(bankServerIPV4)
	if err != nil {
		return "GRPC client not found, debit failed", err
	}
	ctx := context.Background()
	if err := debitLimiter.Wait(ctx); err != nil {
		return "Rate limit exceeded", err
	}

	jsonString, err := json.Marshal(data)

	res, err := client.UnarryCall(ctx, &pb.Clientmsg{Name: string(jsonString)})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
		return "Debit Request failed", err
	}
	log.Printf("success debit: %s", res.GetMessage())
	return res.GetMessage(), nil
}

func resolveRequest(resolverServerIPV4 string, data RequestDataResolver) (ReplyDataResolver, error) {
	// log the data it receives
	// log.Printf("Data: %v", data)
	client, err := getGRPCClientResolver(resolverServerIPV4)
	if err != nil {
		return ReplyDataResolver{}, err
	}
	ctx := context.Background()
	if err := resolveLimiter.Wait(ctx); err != nil {
		return ReplyDataResolver{}, err
	}

	jsonString, err := json.Marshal(data)
	if err != nil {
		return ReplyDataResolver{}, err
	}
	// log the marshalled data
	r, err := client.UnarryCall(ctx, &resolverpb.Clientmsg{Name: string(jsonString)})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
		// return empty ReplyDataResolver and error
		return ReplyDataResolver{}, err
	}
	var replyData ReplyDataResolver
	err = json.Unmarshal([]byte(r.GetMessage()), &replyData.Responses)
	if err != nil {
		return ReplyDataResolver{}, err
	}
	return replyData, nil
}

func CreditRequest(bankServerIPV4 string, data RequestDataBank) (string, error) {
	client, err := getGRPCClient(bankServerIPV4)
	if err != nil {
		return "GRPC client not found, credit failed", err
	}
	ctx := context.Background()
	jsonString, err := json.Marshal(data)

	r, err := client.UnarryCall(ctx, &pb.Clientmsg{Name: string(jsonString)})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
		return "", err
	}
	log.Printf("Greeting: %s", r.GetMessage())
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
	file, err := os.OpenFile("Failed_Transaction.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("%s", err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write(data); err != nil {
		log.Fatalf(err.Error())
	}
}

func generateTransactionID() string {

	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(100000000))
}

func TransferHandler(c echo.Context) error {
	// Get the resolver server address

	resolverServerIPV4 := config.ResolverServerIPV4 + ":" + config.ResolverServerPort
	// Get the bank server address
	debitBankServerIPV4 := config.DebitBankServerIPV4 + ":" + config.DebitBankServerPort
	// Get the bank server address
	creditBankServerIPV4 := config.CreditBankServerIPV4 + ":" + config.CreditBankServerPort

	// Create an empty RequestDataResolver
	var resolveData RequestDataResolver

	// Bind the incoming JSON to resolveData
	if err := c.Bind(&resolveData); err != nil {
		return err
	}

	tid := generateTransactionID()
	resolveData.Requests[0].TransactionID = tid
	resolveData.Requests[1].TransactionID = tid
	// Resolve PaymentIDs
	replyData, err := resolveRequest(resolverServerIPV4, resolveData)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Resolve Failed: %v", err))
	}
	log.Printf("Resolver Response: %v", replyData)
	var replyResolverPayer ReplyResolver
	var replyResolverPayee ReplyResolver
	for _, ReplyResolver := range replyData.Responses {
		if ReplyResolver.PaymentID == resolveData.Requests[0].PaymentID {
			replyResolverPayer = ReplyResolver
		}
		if ReplyResolver.PaymentID == resolveData.Requests[1].PaymentID {
			replyResolverPayee = ReplyResolver
		}
	}
	log.Printf("Payer: %v", replyResolverPayer)
	log.Printf("Payee: %v", replyResolverPayee)

	// check if the account number is not found for any of the parties
	if replyResolverPayer.Status == "not found" {
		return c.String(http.StatusBadRequest, "Payer Account not found")
	}
	if replyResolverPayee.Status == "not found" {
		return c.String(http.StatusBadRequest, "Payee Account not found")
	}

	// check if the account number is found for both
	if replyResolverPayer.Status == "found" && replyResolverPayee.Status == "found" {
		log.Printf("Payer: %s", replyResolverPayer.AccountNumber)
		log.Printf("Payee: %s", replyResolverPayee.AccountNumber)
		debitData := RequestDataBank{
			TransactionID: replyResolverPayer.TransactionID,
			AccountNumber: replyResolverPayer.AccountNumber,
			IFSCCode:      replyResolverPayer.IFSCCode,
			HolderName:    replyResolverPayer.HolderName,
			Amount:        100,
			Type:          "debit",
		}
		creditData := RequestDataBank{
			TransactionID: replyResolverPayee.TransactionID,
			AccountNumber: replyResolverPayee.AccountNumber,
			IFSCCode:      replyResolverPayee.IFSCCode,
			HolderName:    replyResolverPayee.HolderName,
			Amount:        100,
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

				dumpTranscation(replyResolverPayee.TransactionID, debitBankServerIPV4, creditBankServerIPV4, "debit", debitData.Amount, replyResolverPayee.AccountNumber, replyResolverPayee.HolderName, replyResolverPayee.IFSCCode, replyResolverPayer.AccountNumber, replyResolverPayer.HolderName, replyResolverPayer.IFSCCode)
				return c.String(http.StatusInternalServerError, "Debit Failed")
			}
			_, err_ := CreditRequest(creditBankServerIPV4, creditData)
			if err_ != nil {

				msg, _ := creditRetry(creditBankServerIPV4, creditData)
				if msg == "Failed" {
					dumpTranscation(replyResolverPayee.TransactionID, debitBankServerIPV4, creditBankServerIPV4, "credit", creditData.Amount, replyResolverPayee.AccountNumber, replyResolverPayee.HolderName, replyResolverPayee.IFSCCode, replyResolverPayer.AccountNumber, replyResolverPayer.HolderName, replyResolverPayer.IFSCCode)
					return c.String(http.StatusInternalServerError, "Credit Failed")
				}
				//return c.String(http.StatusInternalServerError, "Credit Failed")
			}
			return c.String(http.StatusOK, "Transfer Successful")
		}
		log.Printf("Debit: %s", x)
		_, err_ := CreditRequest(creditBankServerIPV4, creditData)
		if err_ != nil {

			msg, _ := creditRetry(creditBankServerIPV4, creditData)
			if msg == "Failed" {
				dumpTranscation(replyResolverPayee.TransactionID, debitBankServerIPV4, creditBankServerIPV4, "credit", creditData.Amount, replyResolverPayee.AccountNumber, replyResolverPayee.HolderName, replyResolverPayee.IFSCCode, replyResolverPayer.AccountNumber, replyResolverPayer.HolderName, replyResolverPayer.IFSCCode)
				return c.String(http.StatusInternalServerError, "Credit Failed")
			}
			//return c.String(http.StatusInternalServerError, "Credit Failed")
		}
		successfulRequests++
		log.Printf("Successful Requests: %d", successfulRequests)
		return c.String(http.StatusOK, "Transfer Successful")
	}
	return c.String(http.StatusInternalServerError, "Transfer Failed")
}
