#!/bin/bash

docker run -p 5001:50 bank:latest&

docker run -p 5002:50 bank:latest&

docker run -p 5003:50 bank:latest&