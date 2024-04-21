#!/bin/bash

url="http://localhost:80/transfer"  # Replace with your server URL
total_requests=1000 # Total number of requests
duration=600 # Duration in seconds
threads=4  # Number of threads

# Generate a Lua script for wrk
lua_script="wrk_script.lua"
echo "wrk.method = 'POST'" > $lua_script
echo "wrk.headers['Content-Type'] = 'application/json'" >> $lua_script
echo "counter = 0" >> $lua_script
echo "request = function()" >> $lua_script
echo "  counter = counter + 1" >> $lua_script
echo "  local body = [[
{
    \"Requests\":[
        {
            \"TransactionID\": \"]] .. counter .. [[\",
            \"PaymentID\": \"1\",
            \"Type\": \"resolve\"
        },
        {
            \"TransactionID\": \"]] .. counter .. [[\",
            \"PaymentID\": \"2\",
            \"Type\": \"resolve\"
        }
    ]
}
]]" >> $lua_script
echo "  return wrk.format(nil, nil, nil, body)" >> $lua_script
echo "end" >> $lua_script

# Send concurrent requests with custom data using wrk
wrk -c $total_requests -d ${duration}s -t $threads -s $lua_script $url