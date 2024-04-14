#!/bin/bash

# Get the private IP address of the computer
private_ip=$(ifconfig | grep 'inet ' | grep -v '127.0.0.1' | awk '{print $2}' | head -n 1)

# Define the upstream servers with the private IP address
upstream_servers="server $private_ip:8001;
server $private_ip:8002;
server $private_ip:8003;"

# Create the configuration file dynamically
cat <<EOF > lb.conf
events {
    worker_connections 1024;
}

http {
    upstream backend {
        $upstream_servers
    }

    server {
        listen 80;

        location / {
            proxy_pass http://backend;
            proxy_set_header Host \$host;
            proxy_set_header X-Real-IP \$remote_addr;
            proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto \$scheme;

            # kill cache
            add_header Last-Modified \$date_gmt;
            add_header Cache-Control 'no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0';
            if_modified_since off;
            expires off;
            etag off;
        }
    }
}
EOF
