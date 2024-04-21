#!/bin/bash

url="http://localhost:80/"  # Replace with your server URL
total_requests=10000 # Total number of requests
duration=100  # Duration in seconds

# Generate a Lua script for wrk
lua_script="wrk_script.lua"
echo "wrk.method = 'GET'" > $lua_script
echo "wrk.headers['Content-Type'] = 'application/json'" >> $lua_script
echo "request = function()" >> $lua_script
echo "  return wrk.format(nil, url)" >> $lua_script
echo "end" >> $lua_script

# Send concurrent requests with custom data using wrk
wrk -c $total_requests -d ${duration}s -t 4 -s $lua_script $url
