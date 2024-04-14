## Run three instances of the server
make new build if you have changes
```bash
go build
```

run `start.sh` file
```bash
./start.sh
```

## How to run the nginx server?
generate config file for your private ip
```bash
./make_conf.sh
```
ps: make sure you have proper permissions

docker build the image
```bash
docker build -t loadbalancer:latest .
```

```bash
docker run -p 80:80 loadbalancer:latest
```

run docker image
```bash
docker run -p 8000:8000 tpg:latest
```
ps: `-p 8000:8000` will expose the 8000 port of the docker container outside port `8000`