#!/bin/sh

set -xeo pipefail

if [ -z $DNS_SERVER_IP ];then
echo "miss some env args, like DNS_SERVER_IP"
echo "you should use docker -e DNS_SERVER_IP=LOCAL-SERVER-IP"
exit 1
fi
if [ -z $DATA_PATH ];then export DATA_PATH=/opt/dnswm;fi
if [ -z $DNSWM_PORT ];then export DNSWM_PORT=9001;fi

if [ ! -f ${DATA_PATH}/Corefile ];then
mkdir -p ${DATA_PATH}/zones
cat > ${DATA_PATH}/Corefile <<EOF
.:53 {
  hosts {
    ${DNS_SERVER_IP} ns1.mydns.local
    ttl 60
    reload 1m
    fallthrough
  }
  forward . /etc/resolv.conf
  cache 120
  reload 5s
  log
  errors

}

lan {
  auto {
    directory ${DATA_PATH}/zones (.*) {1}
    reload 5s
  }
}
EOF
else
CHECK_DNS_SERVER_IP=$(grep "ns1.mydns.local" ${DATA_PATH}/Corefile|awk '{print $1}')
if [[ "$CHECK_DNS_SERVER_IP" == "ns1.mydns.local" ]]
then
  echo "MISS SOME ENV VAR, LIKE DNS_SERVER_IP"
  echo "YOU SHOULD DELETE THIS CONTAINER, AND USE DOKCER -e DNS_SERVER_IP='YOUR_IP' TO CREATE NEW CONTAINER AGAIN"
  exit 1
fi
fi

if [ ! -f /etc/supervisor.d/dnswm.ini ];then
mkdir /etc/supervisor.d
cat >/etc/supervisor.d/dnswm.ini <<EOF
[program:coredns]
command =./coredns -conf ${DATA_PATH}/Corefile
autostart=true
autorestart=true
priority=5
stdout_events_enabled=true
stderr_events_enabled=true
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0
stopsignal=QUIT

[program:dnswm]             
command=./dnswm -p $DNSWM_PORT -d $DATA_PATH
autostart=true
autorestart=true
priority=10
stdout_events_enabled=true
stderr_events_enabled=true
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0
stopsignal=QUIT
EOF
fi

# Start supervisord and services
exec /usr/bin/supervisord -n -c /etc/supervisord.conf

