#!/bin/bash

# Get the private IP address of the computer
#private_ip=$(ifconfig | grep 'inet ' | grep -v '127.0.0.1' | awk '{print $2}' | head -n 1)
private_ip=10.240.1.252
# Define the upstream servers with the private IP address
upstream_servers="server $private_ip:3001;
            server $private_ip:3002;
            server $private_ip:3003;"

# Create the configuration file dynamically
cat <<EOF > lb.conf
events {
    worker_connections 1024;
}

http {
    upstream grpcservers {
        $upstream_servers
    }

    server {
        listen 30 http2;
  
        location / {
            # Replace localhost:50051 with the address and port of your gRPC server
            # The 'grpc://' prefix is optional; unencrypted gRPC is the default
            grpc_pass grpc://grpcservers;


            # kill cache
            add_header Last-Modified $date_gmt;
            add_header Cache-Control 'no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0';
            if_modified_since off;
            expires off;
            etag off;
        }
    }
}
EOF
