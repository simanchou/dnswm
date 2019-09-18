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


### API
###### domain
query domain
```
curl -s -XGET http://192.168.2.98:9001/api/domain|jq
{
  "Code": 0,
  "Msg": "query all domains done",
  "Data": [
    {
      "Name": "t1.lan",
      "CreatedAt": "2019-09-18 17:09:08"
    },
    {
      "Name": "t2.lan",
      "CreatedAt": "2019-09-18 17:09:11"
    },
    {
      "Name": "t3.lan",
      "CreatedAt": "2019-09-18 17:09:14"
    },
    {
      "Name": "t4.lan",
      "CreatedAt": "2019-09-18 17:09:35"
    },
    {
      "Name": "t5.lan",
      "CreatedAt": "2019-09-18 17:09:38"
    }
  ]
}
```
add domain

```
curl -s -XPOST -d 'domain=t6.lan' http://192.168.2.98:9001/api/domain|jq
{
  "Code": 0,
  "Msg": "domain t6.lan add successful",
  "Data": null
}

```

delete domain

```
curl -s -XDELETE http://192.168.2.98:9001/api/domain?domain=t4.lan|jq
{
  "Code": 0,
  "Msg": "domain t4.lan delete successful",
  "Data": null
}
```
###### record
add record

```
curl -s -XPOST -d 'domain=t5.lan&name=w4&type=A&ttl=600&value=4.4.4.4' http://192.168.2.98:9001/api/record|jq
{
  "Code": 0,
  "Msg": "add record [ w4 A 4.4.4.4 ] for domain t5.lan successful",
  "Data": null
}

```
query record

```
curl -s -XGET http://192.168.2.98:9001/api/record?domain=t5.lan|jq
{
  "Code": 0,
  "Msg": "",
  "Data": {
    "Name": "t5.lan",
    "Serial": 7,
    "Records": {
      "3584b32adb4a401292c5fcd0b397e4ef": {
        "ID": "3584b32adb4a401292c5fcd0b397e4ef",
        "Name": "w1",
        "Type": "A",
        "TTL": 600,
        "Priority": -1,
        "Value": "1.1.1.1"
      },
      "a17b751e19c93511a4d72f28886e5dd2": {
        "ID": "a17b751e19c93511a4d72f28886e5dd2",
        "Name": "w4",
        "Type": "A",
        "TTL": 600,
        "Priority": -1,
        "Value": "4.4.4.4"
      },
      "bf0c22289f180dd2e0a2e3c6d38bd6e1": {
        "ID": "bf0c22289f180dd2e0a2e3c6d38bd6e1",
        "Name": "w3",
        "Type": "A",
        "TTL": 600,
        "Priority": -1,
        "Value": "3.3.3.3"
      },
      "cde6b10d0e5c88aad7c6f7ff0a89fb33": {
        "ID": "cde6b10d0e5c88aad7c6f7ff0a89fb33",
        "Name": "w2",
        "Type": "A",
        "TTL": 600,
        "Priority": -1,
        "Value": "2.2.2.2"
      }
    },
    "CreatedAt": "2019-09-18 17:09:38"
  }
}


```
delete record
> the id witch in the url is from query record, "ID": "a17b751e19c93511a4d72f28886e5dd2"

```
curl -s -XDELETE 'http://192.168.2.98:9001/api/record?domain=t5.lan&id=a17b751e19c93511a4d72f28886e5dd2'|jq
{
  "Code": 0,
  "Msg": "delete record for domain t5.lan successful",
  "Data": null
}
```
