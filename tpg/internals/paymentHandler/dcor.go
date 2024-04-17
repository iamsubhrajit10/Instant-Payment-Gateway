package paymenthandler

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"time"
	"tpg/config"
	pb "tpg/protos"

	"github.com/labstack/echo/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	GrpcClientMap     = make(map[string]pb.DetailsClient)
	GrpcConnectionMap = make(map[string]*grpc.ClientConn)
)

func getGRPCConnection(address string) (*grpc.ClientConn, error) {
	addr := flag.String("addr", address, "the address to connect to")
	if _, ok := GrpcConnectionMap[*addr]; !ok {
		conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			config.Logger.Fatalf("did not connect: %v", err)
			return nil, err
		}
		GrpcConnectionMap[*addr] = conn
	}
	return GrpcConnectionMap[*addr], nil
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

type RequestData struct {
	TransactionID string
	AccountNumber string
	Amount        int
	Type          string
}

func debitRequest(bankServerIPV4 string, data RequestData) (string, error) {

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
	return "Success", nil
}

func creditRequest(bankServerIPV4 string, data RequestData) (string, error) {
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

func debitRetry(addr string, data RequestData) (string, error) {
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
	//time.Sleep(1 * time.Second)
	debitBankServerIPV4 := config.DebitBankServerIPV4 + ":" + config.DebitBankServerPort
	creitBankServerIPV4 := config.CreditBankServerIPV4 + ":" + config.CreditBankServerPort
	debitData := RequestData{
		TransactionID: "1",
		AccountNumber: "1234",
		Amount:        100,
		Type:          "debit",
	}

	creditData := RequestData{
		TransactionID: "1",
		AccountNumber: "3993",
		Amount:        100,
		Type:          "credit",
	}
	_, err := debitRequest(debitBankServerIPV4, debitData)
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
	_, err_ := creditRequest(creitBankServerIPV4, creditData)

	if err_ != nil {
		return c.String(http.StatusInternalServerError, "Credit Failed")
	}
	return c.String(http.StatusOK, "Transfer Successful")
}
