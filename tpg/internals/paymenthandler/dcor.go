package paymenthandler

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
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

func resolveRequest(resolverServerIPV4 string, data RequestDataResolver) (ReplyDataResolver, error) {
	// log the data it receives
	// log.Printf("Data: %v", data)
	client, err := getGRPCClientResolver(resolverServerIPV4)
	if err != nil {
		return ReplyDataResolver{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

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
		x, err := debitRequest(debitBankServerIPV4, debitData)
		if err != nil {
			msg, _ := debitRetry(debitBankServerIPV4, debitData)
			if msg == "Failed" {
				return c.String(http.StatusInternalServerError, "Debit Failed")
			}
			_, err_ := creditRequest(creditBankServerIPV4, creditData)
			if err_ != nil {
				return c.String(http.StatusInternalServerError, "Credit Failed")
			}
			return c.String(http.StatusOK, "Transfer Successful")
		}
		log.Printf("Debit: %s", x)
		_, err_ := creditRequest(creditBankServerIPV4, creditData)

		if err_ != nil {
			return c.String(http.StatusInternalServerError, "Credit Failed")
		}
		return c.String(http.StatusOK, "Transfer Successful")
	}
	return c.String(http.StatusInternalServerError, "Transfer Failed")
}
