# dnswm
Dns web manager is an out of the box tool for dns server and web manager(implement web gui and api for [coredns](https://github.com/coredns/coredns)).

### Install
1. use docker to try

```
export DNS_SERVER_IP=$(hostname -I | cut -d" " -f 1)
docker run --name dnswm --rm -p53:53/udp -p9001:9001 -e DNS_SERVER_IP='$DNS_SERVER_IP' dnswm
```
> Attention: data won't save by the example above!
2. use docker-compose

```
version: '2'
services:
  dnswm:
    container_name: dnswm
    image: simanchou/dnswm
    restart: always
    environment:
      - DNS_SERVER_IP=192.168.2.98  #change your ip here
    ports:
      - "53:53/udp"
      - "9001:9001"
    volumes:
      - ./data/:/opt/dnswm/
      - /etc/localtime:/etc/localtime
```

### GUI
![image](https://github.com/simanchou/dnswm/blob/master/example/01.png)

![image](https://github.com/simanchou/dnswm/blob/master/example/02.png)
