## How to run the nginx server?
generate config file for your private ip
```bash
./make_conf.sh
```
ps: make sure you have proper permissions

docker build the image
```bash
docker build -t resolverlb:latest .
```
run the container with the image made
```bash
docker run -p 30:30 resolverlb:latest
```