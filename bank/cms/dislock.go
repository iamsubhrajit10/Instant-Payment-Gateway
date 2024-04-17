// a centralized mutex server implementation
// dislock implementation
package cms

import (
	msgp "bank/cms/msg"
	netq "bank/cms/netq"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	// "os"
	// "sync"
)

type dislock struct {
	cli    netq.Client
	lockID int // regard it as process id.
	port   int
	logger *log.Logger

	// sata info
	readCnt  int
	writeCnt int
}

func NewDislock(port, lockID int) (*dislock, error) {
	dl := &dislock{port: port, lockID: lockID}
	dl.logger = CreateLog("log/dislock_"+strconv.Itoa(lockID)+".log", "[dislock] ")
	cli, err := netq.NewClient(dl.port)
	if err != nil {
		dl.logger.Printf("dislock(%v) create error: %v.\n", dl.lockID, err.Error())
		return nil, err
	}
	dl.cli = cli
	return dl, nil
}

// TODO: handle timeout.
func (dl *dislock) Acquire(receiver int, msgContent interface{}) error {
	// send lock request message.
	lr := msgp.NewRequest(dl.lockID, receiver, msgContent)
	log.Printf("lock(%v) send request lock message(%v) to server.\n", dl.lockID, lr.String())
	lrBytes, _ := json.Marshal(lr)
	log.Printf("lock(%v) send request lock message(%v) to server.\n", dl.lockID, lr.String())
	if err := dl.cli.WriteData(lrBytes); err != nil {
		log.Printf("lock(%v) send request message(%v) error: %v.\n", dl.lockID, lr.String(), err.Error())
		return err
	}
	dl.writeCnt++
	// wait for lock grant message
	log.Printf("lock(%v) wait grant message from server.\n", dl.lockID)
	lgBytes, err := dl.cli.ReadData()
	if err != nil {
		dl.logger.Printf("lock(%v) receive Grant message error: %v.\n", dl.lockID, err.Error())
		return err
	}
	dl.readCnt++
	var lg msgp.Message
	json.Unmarshal(lgBytes, &lg)
	if lg.MsgType == msgp.Grant {
		// neglect message content.
		dl.logger.Printf("lock(%v) receive Grant message(%v) from server.\n", dl.lockID, lg.String())
		return nil
	} else {
		errMsg := fmt.Sprintf("lock(%v) receive error message(%v) from server.\n", dl.lockID, lg.String())
		dl.logger.Printf(errMsg)
		return errors.New(errMsg)
	}
}

func (dl *dislock) Release(receiver int, msgContent interface{}) error {
	// send lock release message.
	lrl := msgp.NewRelease(dl.lockID, receiver, msgContent)
	lrlBytes, _ := json.Marshal(lrl)
	if err := dl.cli.WriteData(lrlBytes); err != nil {
		dl.logger.Printf("lock(%v) send release message(%v) error: %v.\n", dl.lockID, lrl.String(), err.Error())
		return err
	}
	dl.writeCnt++
	dl.logger.Printf("lock(%v) send release message(%v) successfully.\n", dl.lockID, lrl.String())
	// dl.cli.Close() // close connection
	// dl.logger.Printf("lock(%v) closed successfully.\n", dl.lockID)
	return nil
}

// @see process.Close
func (dl *dislock) Close() error {
	if err := dl.cli.Close(); err != nil {
		return err
	}
	return nil
}
