## How to run three instances of transaction-processing-gateway?

get docker image ready
```bash
docker build -t tpg:latest .
```
run three instances of this container
```bash
./duplicate_container.sh
```
