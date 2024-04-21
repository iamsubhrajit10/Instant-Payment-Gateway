#!/bin/bash

# Install pv if not already available (adjust package manager based on your system)
# sudo apt install pv  # For Debian/Ubuntu based systems

# The URL of your server
url="http://localhost:8000/transfer"

# The total number of transactions
total_transactions=100000

# The number of transactions per request
transactions_per_request=2

# The total number of requests
total_requests=$((total_transactions / transactions_per_request))

# The Content-Type header for the requests
content_type="Content-Type: application/json"

# Get the start time
start_time=$(date +%s)

# Limit rate to 1000 requests per second with pv
seq 1 $total_requests | pv -L 4000 | xargs -I {} -P 4000 bash -c '
    i=$1
    payment_id_1=$((2*i-1))
    payment_id_2=$((2*i))
    data="{\"Requests\":[{\"TransactionID\":\"$i\",\"PaymentID\":\"$payment_id_1\",\"Type\":\"resolve\"},{\"TransactionID\":\"$i\",\"PaymentID\":\"$payment_id_2\",\"Type\":\"resolve\"}]}"
    response=$(curl -s -X POST -H "'"$content_type"'" -d "$data" "'"$url"'")
    if [ -z "$response" ]; then
        echo -e "Count: $i, Empty response" >> output_test_2K.txt
        exit 1
    fi
    echo -e "Count: $i, Response: $response\n" >> output_test_2K.txt
' _ {}

# Get the end time
end_time=$(date +%s)

# Calculate the elapsed time
elapsed_time=$((end_time - start_time))

# Print the elapsed time
echo "Total elapsed time: $elapsed_time seconds" >> output_test_2K.txt