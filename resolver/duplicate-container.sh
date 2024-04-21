#!/bin/bash

docker run -p 3001:30 resolver:latest&

docker run -p 3002:30 resolver:latest&

docker run -p 3003:30 resolver:latest&