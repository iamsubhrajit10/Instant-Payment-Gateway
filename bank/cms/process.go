package cms

import (
	"bank/config"

	_ "github.com/go-sql-driver/mysql"
)

type RequestData struct {
	TransactionID string
	AccountNumber string
	Amount        int
	Type          string
}

type process struct {
	dl            *dislock
	port          int
	Type          string
	accountNumber string
}

func NewProcess(port int, accountNumber string, Type string) (*process, error) {
	p := &process{port: port, accountNumber: accountNumber, Type: Type}
	dl, err := NewDislock(port)
	if err != nil {
		config.Logger.Printf(" Account Number :- (%v) create error: %v.\n", accountNumber, err.Error())
		return nil, err
	}
	p.dl = dl
	return p, nil
}

func (p *process) work(data RequestData) (string, error) {
	switch data.Type {
	case "debit":
		{
			config.Logger.Printf("Processing debit request: %v", data.AccountNumber)
			err_ := config.DB.Ping()
			if err_ != nil {
				config.Logger.Printf("Error connecting to the database: %v", err_)
				msg, err := config.ConnectWithSql()
				if err != nil {
					return msg, err
				}
			}
			results, err := config.DB.Query("SELECT Amount FROM bank_details WHERE Account_number = ?", data.AccountNumber)
			if err != nil {
				config.Logger.Fatal(err)
				return "", err
			}

			for results.Next() {
				var amount int
				// for each row, scan the result into our tag composite object
				err = results.Scan(&amount)
				if err != nil {
					config.Logger.Fatal(err)
					return "", err
				}
				if amount < data.Amount {
					config.Logger.Printf("Insufficient balance")
					return "Insufficient balance", nil
				} else {
					amount = amount - data.Amount
					_, err := config.DB.Exec("UPDATE bank_details SET Amount = ? WHERE Account_number = ?", amount, data.AccountNumber)
					if err != nil {
						config.Logger.Fatal(err)
						return "", err
					}
					return "", nil
				}
			}

			return "No Records Found", nil
		}

	case "credit":
		{
			config.Logger.Printf("Processing credit request: %v", data.AccountNumber)
			err_ := config.DB.Ping()
			if err_ != nil {
				config.Logger.Printf("Error connecting to the database: %v", err_)
				msg, err := config.ConnectWithSql()
				if err != nil {
					return msg, err
				}
			}
			// handle error

			results, err := config.DB.Query("SELECT Amount FROM bank_details WHERE Account_number = ?", data.AccountNumber)
			if err != nil {
				config.Logger.Fatal(err)
				return "", err
			}

			for results.Next() {
				var amount int
				err = results.Scan(&amount)
				if err != nil {
					config.Logger.Fatal(err)
					// proper error handling instead of panic in your app
					return "", err
				}
				amount = amount + data.Amount
				_, err := config.DB.Exec("UPDATE bank_details SET Amount = ? WHERE Account_number = ?", amount, data.AccountNumber)
				if err != nil {
					config.Logger.Fatal(err)
					return "", err
				}
				return "", nil
			}
			return "No Records Found", nil
		}
	}
	return "", nil
}

func (p *process) Run(accountNumber string, Type string, data RequestData) (string, error) {

	config.Logger.Printf("request data: %v\n", data)
	var err error
	err = p.dl.Acquire(accountNumber, Type) // if any process still in critical section, it will block.
	if err != nil {
		config.Logger.Printf("(%v) fail to acquire lock for type (%v): %v.\n", accountNumber, Type, err.Error())
		return "", err
	}

	config.Logger.Printf("Account Number (%v) entered the critical section for %v.\n", accountNumber, Type)
	msg, err := p.work(data) // ignore any failure occurs in this stage temporaily.
	err_ := p.dl.Release(accountNumber, Type)
	if err_ != nil {
		config.Logger.Printf("(%v) fail to release lock of type  %v: %v.\n", accountNumber, Type, err.Error())
		return "", err_
	}

	if err != nil {
		return "", err
	}
	if data.Type == "debit" {
		if msg == "Insufficient balance" {
			return msg, nil
		}
		if msg == "No Records Found" {
			return msg, nil
		}
		return "Debit Success", nil
	}
	if data.Type == "credit" {
		if msg == "No Records Found" {
			return msg, nil
		}
		return "Credit Success", nil
	}

	config.Logger.Printf("(%v) exited the critical section.\n", accountNumber)
	return "", nil
}

// the method is not good in usage logical, because the lock will automatically close when process called Release.
// so it just mainly facilitate our testing cases.
func (p *process) Close() error {
	if err := p.dl.Close(); err != nil {
		return err
	}
	return nil
}
