// a centralized mutex server implementation
// dislock implementation
package cms

import (
	msgp "bank/cms/msg"
	netq "bank/cms/netq"
	"bank/config"
	"encoding/json"
	"errors"
	"fmt"
	//	"fmt"
	//	"log"
	// "os"
	// "sync"
)

type dislock struct {
	cli  netq.Client
	port int
}

func NewDislock(port int) (*dislock, error) {
	dl := &dislock{port: port}
	//dl.logger = CreateLog("log/dislock_"+strconv.Itoa(lockID)+".log", "[dislock] ")
	cli, err := netq.NewClient(dl.port)
	if err != nil {
		//dl.logger.Printf("dislock(%v) create error: %v.\n", dl.lockID, err.Error())
		return nil, err
	}
	dl.cli = cli
	return dl, nil
}

// TODO: handle timeout.
func (dl *dislock) Acquire(accountNumber, Type string) error {
	lr := msgp.NewRequest(accountNumber, Type)
	config.Logger.Printf("%v send request lock message for (%v) to server.\n", accountNumber, Type)
	lrBytes, _ := json.Marshal(lr)
	if err := dl.cli.WriteData(lrBytes); err != nil {
		config.Logger.Printf("%v send request lock message for (%v) to server giving error: (%v).\n", accountNumber, Type, err.Error())
		return err
	}
	config.Logger.Printf("(%v) wait grant message from server.\n", accountNumber)
	lgBytes, err := dl.cli.ReadData()
	if err != nil {
		config.Logger.Printf("(%v) receive Grant message error: %v.\n", accountNumber, err.Error())
		return err
	}
	var lg msgp.Message
	json.Unmarshal(lgBytes, &lg)
	if lg.MsgType == "Grant" {
		config.Logger.Printf("(%v) receive Grant message for (%v) from server.\n", accountNumber, Type)
		return nil
	} else {
		errMsg := fmt.Sprintf("(%v) receive error message for (%v) from server.\n", accountNumber, Type)
		config.Logger.Printf(errMsg)
		return errors.New(errMsg)
	}
}

func (dl *dislock) Release(accountNumber, Type string) error {
	// send lock release message.
	lrl := msgp.NewRelease(accountNumber, Type)
	lrlBytes, _ := json.Marshal(lrl)
	if err := dl.cli.WriteData(lrlBytes); err != nil {
		config.Logger.Printf("(%v) send release message of type (%v) error: %v.\n", accountNumber, Type, err.Error())
		return err
	}
	config.Logger.Printf("(%v) send release message of type (%v) successfully.\n", accountNumber, Type)
	//dl.cli.Close() // close connection
	config.Logger.Printf("(%v) closed successfully.\n", accountNumber)
	return nil
}

// @see process.Close
func (dl *dislock) Close() error {
	if err := dl.cli.Close(); err != nil {
		return err
	}
	return nil
}
