#!/bin/bash

url="http://10.240.2.146:8088/transfer"  # Replace with your server URL
total_requests=1000 # Total number of requests
duration=10 # Duration in seconds
threads=4  # Number of threads

# Generate a Lua script for wrk
lua_script="wrk_script.lua"
echo "wrk.method = 'POST'" > $lua_script
echo "wrk.headers['Content-Type'] = 'application/json'" >> $lua_script
echo "counter = 0" >> $lua_script
echo "request = function()" >> $lua_script
echo "  counter = counter + 1" >> $lua_script
echo "  local paymentId1 = (counter * 2) - 1" >> $lua_script
echo "  local paymentId2 = counter * 2" >> $lua_script
echo "  local body = [[
{
    \"Requests\":[
        {
            \"TransactionID\": \"]] .. counter .. [[\",
            \"PaymentID\": \"]] .. paymentId1 .. [[\",
            \"Type\": \"resolve\"
        },
        {
            \"TransactionID\": \"]] .. counter .. [[\",
            \"PaymentID\": \"]] .. paymentId2 .. [[\",
            \"Type\": \"resolve\"
        }
    ]
}
]]" >> $lua_script
echo "  return wrk.format(nil, nil, nil, body)" >> $lua_script
echo "end" >> $lua_script

# Send concurrent requests with custom data using wrk
wrk -c $total_requests -d ${duration}s -t $threads -s $lua_script $url