#!/bin/bash

sqlite3 bank_details.db <<EOF
CREATE TABLE IF NOT EXISTS bank_details(
    PaymentID INTEGER PRIMARY KEY,
    AccountNumber TEXT NOT NULL,
    IFSCCode TEXT NOT NULL,
    HolderName TEXT NOT NULL
);

DELETE FROM bank_details;

BEGIN TRANSACTION;
$(for i in $(seq 1 100000); do
    echo "INSERT INTO bank_details (PaymentID, AccountNumber, IFSCCode, HolderName) VALUES ($i, '1234567890$i', 'IFSC000$i', 'Holder$i');"
done)
COMMIT;
EOF