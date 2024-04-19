mysql --user=root --password=root -h 172.20.112.1 -P 3306  upi <<EOF
BEGIN;
$(for i in $(seq 1 1000); do
    echo "INSERT INTO bank_details (PaymentID, AccountNumber, IFSCCode, HolderName,Amount) VALUES ('$i', '1234567890$i', 'IFSC000$i', 'Holder$i',$i*1000);"
done)
COMMIT;
EOF