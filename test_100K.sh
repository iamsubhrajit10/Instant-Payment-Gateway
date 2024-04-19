#!/bin/bash

# The URL of your server
url="http://localhost:8000/transfer"

# The total number of transactions
total_transactions=50000

# The number of transactions per request
transactions_per_request=2

# The total number of requests
total_requests=$((total_transactions / transactions_per_request))

# The Content-Type header for the requests
content_type="Content-Type: application/json"

# Send the requests
for ((i=1; i<=total_requests; i++)); do
  # Calculate the PaymentID for the first and second transaction in this request
  payment_id_1=$((2*i-1))
  payment_id_2=$((2*i))

  # The data for the request
  data="{\"Requests\":[{\"TransactionID\":\"$i\",\"PaymentID\":\"$payment_id_1\",\"Type\":\"resolve\"},{\"TransactionID\":\"$i\",\"PaymentID\":\"$payment_id_2\",\"Type\":\"resolve\"}]}"

  # Send the request in the background
  curl -X POST -H "$content_type" -d "$data" "$url" &
done

# Wait for all background jobs to finish
wait