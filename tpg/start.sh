#!/bin/bash

# Start the first instance on port 8001
./tpg 8001 &

# Start the second instance on port 8002
./tpg 8002 &

# Start the third instance on port 8003
./tpg 8003 &

# Wait for all instances to finish before exiting
wait