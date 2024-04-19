#!/bin/bash

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

# Send the requests
for ((i=1; i<=total_requests; i++)); do
    # Calculate the PaymentID for the first and second transaction in this request
    payment_id_1=$((2*i-1))
    payment_id_2=$((2*i))

    # The data for the request
    data="{\"Requests\":[{\"TransactionID\":\"$i\",\"PaymentID\":\"$payment_id_1\",\"Type\":\"resolve\"},{\"TransactionID\":\"$i\",\"PaymentID\":\"$payment_id_2\",\"Type\":\"resolve\"}]}"

    # Send the request and get the response in the background
    {
        response=$(curl -s -X POST -H "$content_type" -d "$data" "$url")

        # Check if the response is empty
        if [ -z "$response" ]; then
            echo -e "Count: $i, Empty response" >> output_test_2K.txt
            exit 1
        fi

        # Print the count and the response
        echo -e "Count: $i, Response: $response\n" >> output_test_2K.txt
    } &
done

# Wait for all background jobs to finish
wait

# Get the end time
end_time=$(date +%s)

# Calculate the elapsed time
elapsed_time=$((end_time - start_time))

# Print the elapsed time
echo "Total elapsed time: $elapsed_time seconds" >> output_test_2K.txt