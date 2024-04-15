package paymenthandler

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"time"
	"tpg/config"
	pb "tpg/protos"
	resolverpb "tpg/resolverproto"
	"github.com/labstack/echo/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"fmt"
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
	IFSCCode string
	HolderName string
	Amount        int
	Type          string
}

type RequestDataResolver struct {
	TransactionID string
	PaymentID	 string
	Type 		string
}
type ReplyDataResolver struct {	
    TransactionID  string `json:"TransactionID"`
    PaymentID      string `json:"PaymentID"`
    Status         string `json:"Status"`
    AccountNumber  string `json:"AccountNumber"`
    IFSCCode       string `json:"IFSCCode"`
    HolderName     string `json:"HolderName"`
}

func getGRPCConnection(address string) (*grpc.ClientConn, error) {
	addr := flag.String("addr", address, "the address to connect to")
	if _, ok := GrpcConnectionMap[*addr]; !ok {
		conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			//log.Fatalf("did not connect: %v", err)
			return nil, err
		}
		GrpcConnectionMap[*addr] = conn
	}
	return GrpcConnectionMap[*addr], nil
}

func getGRPCConnectionResolver(address string) (*grpc.ClientConn, error) {
	// fmt.Println(address)
	if _, ok := GrpcConnectionMapRes[address]; !ok {
		conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("did not connect: %v", err)
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



func debitRequest(bankServerIPV4 string, data RequestDataBank) (string, error) {

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
		log.Fatalf("could not greet: %v", err)
		return "Debit Request failed", err
	}
	log.Printf("success debit: %s", res.GetMessage())
	return "Success", nil
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
		log.Fatalf("could not greet: %v", err)
		return "", err
	}
	log.Printf("Greeting: %s", r.GetMessage())
	return r.GetMessage(), nil
}

func creditRequest(bankServerIPV4 string, data RequestDataBank) (string, error) {
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
		log.Fatalf("could not greet: %v", err)
		return "", err
	}
	log.Printf("Greeting: %s", r.GetMessage())
	return "Success", nil
}

func debitRetry(addr string, data RequestDataBank) (string, error) {
	for i := 0; i < config.DebitRetries; i++ {
		_, err := debitRequest(addr, data)
		if err == nil {
			return "Success", nil
		}
	}
	return "Failed", nil
}

func TransferHandler(c echo.Context) error {
	//reply that i am responsible for transfer
	//ti me.Sleep(1 * time.Second)
	resolverServerIPV4 := config.ResolverServerIPV4 + ":" + config.ResolverServerPort
	println(resolverServerIPV4)
	debitBankServerIPV4 := config.DebitBankServerIPV4 + ":" + config.DebitBankServerPort
	creitBankServerIPV4 := config.CreditBankServerIPV4 + ":" + config.CreditBankServerPort
	resolveDataPayer := RequestDataResolver{
		TransactionID: "1",
		PaymentID: "1",
		Type: "resolve",
	}
	resolveDataPayee := RequestDataResolver{
		TransactionID: "1",
		PaymentID: "2",
		Type: "resolve",
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
		log.Printf("Payer Response: %s", resolverResponsePayer)

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
		log.Printf("Payee Response: %s", resolveResponsePayee)

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
		log.Printf("Payer: %s", replyResolverPayer.AccountNumber)
		log.Printf("Payee: %s", replyResolverPayee.AccountNumber)
		debitData := RequestDataBank{
			TransactionID: replyResolverPayer.TransactionID,
			AccountNumber: replyResolverPayer.AccountNumber,
			IFSCCode: replyResolverPayer.IFSCCode,
			HolderName: replyResolverPayer.HolderName,
			Amount: 100,
			Type: "debit",
		}
		creditData := RequestDataBank{
			TransactionID: replyResolverPayee.TransactionID,
			AccountNumber: replyResolverPayee.AccountNumber,
			IFSCCode: replyResolverPayee.IFSCCode,
			HolderName: replyResolverPayee.HolderName,
			Amount: 100,
			Type: "credit",
		}

		x, err := debitRequest(debitBankServerIPV4, debitData)
		if err != nil {
			msg, _ := debitRetry(debitBankServerIPV4, debitData)
			if msg == "Failed" {
				return c.String(http.StatusInternalServerError, "Debit Failed")
			}
			_, err_ := creditRequest(creitBankServerIPV4, creditData)
			if err_ != nil {
				return c.String(http.StatusInternalServerError, "Credit Failed")
			}
			return c.String(http.StatusOK, "Transfer Successful")
		}
		log.Printf("Debit: %s", x)
		_, err_ := creditRequest(creitBankServerIPV4, creditData)
	
		if err_ != nil {
			return c.String(http.StatusInternalServerError, "Credit Failed")
		}
		return c.String(http.StatusOK, "Transfer Successful")
	}
	return c.String(http.StatusInternalServerError, "Transfer Failed")
}
