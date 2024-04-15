#!/bin/bash

sqlite3 bank_details.db <<EOF
CREATE TABLE bank_details(
    PaymentID INTEGER PRIMARY KEY,
    AccountNumber TEXT NOT NULL,
    IFSCCode TEXT NOT NULL,
    HolderName TEXT NOT NULL
);

INSERT INTO bank_details(PaymentID, AccountNumber, IFSCCode, HolderName) VALUES
(1, '1234567890', 'IFSC0001', 'Aman Gupta'),
(2, '2345678901', 'IFSC0002', 'Chirag Modi'),
(3, '3456789012', 'IFSC0003', 'Subhrajit Das');
EOF