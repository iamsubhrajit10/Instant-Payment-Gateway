mysql --user=root --password=root --host=localhost --port=3306  upi <<EOF
#mysql --user root --password root -h localhost -P 3306  upi <<EOF
BEGIN;

Create table bank_details(
    PaymentID varchar(20) primary key,
    AccountNumber varchar(20),
    IFSCCode varchar(20),
    HolderName varchar(20),
    Amount int
);


$(for i in $(seq 1 1000); do
    if [ $((i % 3)) -eq 2 ]; then
        echo "INSERT INTO bank_details (PaymentID, AccountNumber, IFSCCode, HolderName, Amount) VALUES ('$i', '1234567890$i', 'IFSC000$i', 'Holder$i', $i*1000);"
    fi
done)

COMMIT;
EOF