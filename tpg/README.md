## How to run the server?
build the docker image
```bash
docker build --tag tpg:latest .
```

run docker image
```bash
docker run -p 8000:8000 tpg:latest
```
ps: `-p 8000:8000` will expose the 8000 port of the docker container outside port `8000`