#!/bin/bash

# Get a list of all container IDs
container_ids=$(docker ps -q)

# Loop through each container ID
for container_id in $container_ids; do
    # Get the container name
    container_name=$(docker inspect -f '{{.Name}}' $container_id)
    
    # Remove the leading slash from the container name
    container_name=${container_name:1}

    # Get the container IP address
    container_ip=$(docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $container_id)

    # Print the container name and IP address
    echo "Container: $container_name, IP Address: $container_ip"
done
