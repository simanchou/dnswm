VERSION="1.0"
PKG_MD5="48eaa9371a2571ed62188a6ccc041423"
export DATA_PATH=/opt/dnswm
export DNSWM_PORT=9001
useradd -M -s /sbin/nologin dnswm
mkdir -p ${DATA_PATH}/zones
mkdir -p ${DATA_PATH}/bin

if [[ ! -f /tmp/dnswm-linux-${VERSION}.tar.gz ]];then
    echo "begin to download tar file dnswm-linux-${VERSION}.tar.gz"
    echo "if download fail, try again or download yourself and upload to /tmp/dnswm-linux-${VERSION}.tar.gz"
    curl -L https://github.com/simanchou/dnswm/releases/download/v${VERSION}/dnswm-linux-${VERSION}.tar.gz -o /tmp/dnswm-linux-${VERSION}.tar.gz
else
    echo "dnswm-linux-${VERSION}.tar.gz already exist"
    echo "begin to check md5 of dnswm-linux-${VERSION}.tar.gz"
    EXIST_FILE_MD5=$(md5sum /tmp/dnswm-linux-${VERSION}.tar.gz|awk '{print $1}')
    if [[ "$EXIST_FILE_MD5" == "$PKG_MD5" ]];then
        echo "tar file validate successful"
        echo "begin to install dnswm"
    else
        echo "file validate fail.try again, or download yourself and upload to /tmp/dnswm-linux-${VERSION}.tar.gz"
        echo "download url: https://github.com/simanchou/dnswm/releases/download/v${VERSION}/dnswm-linux-${VERSION}.tar.gz"
        rm -rf /tmp/dnswm-linux-${VERSION}.tar.gz
        exit 1
    fi
fi

tar zxf /tmp/dnswm-linux-${VERSION}.tar.gz -C ${DATA_PATH}/bin
export DNS_SERVER_IP=$(hostname -I | cut -d" " -f 1)

cat >/etc/systemd/system/coredns.service <<EOF
[Unit]
Description=CoreDNS DNS server
Documentation=https://coredns.io
After=network.target

[Service]
PermissionsStartOnly=true
LimitNOFILE=1048576
LimitNPROC=512
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_BIND_SERVICE
NoNewPrivileges=true
User=dnswm
WorkingDirectory=${DATA_PATH}/bin
ExecStart=${DATA_PATH}/bin/coredns -conf=/opt/dnswm/Corefile
ExecReload=/bin/kill -SIGUSR1 $MAINPID
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

cat >/etc/systemd/system/dnswm.service <<EOF
[Unit]
Description=DNS Web Manager
Documentation=https://github.com/simanchou/dnswm
After=network.target

[Service]
PermissionsStartOnly=true
LimitNOFILE=1048576
LimitNPROC=512
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_BIND_SERVICE
NoNewPrivileges=true
User=dnswm
WorkingDirectory=${DATA_PATH}/bin
ExecStart=${DATA_PATH}/bin/dnswm -p $DNSWM_PORT -d $DATA_PATH
ExecReload=/bin/kill -SIGUSR1 $MAINPID
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

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

chown -R dnswm:dnswm ${DATA_PATH}

echo "finish install, start service"
systemctl daemon-reload
systemctl restart coredns
systemctl restart dnswm

systemctl enable coredns
systemctl enable dnswm

DNS_SERVICE_IS_ACTIVE=1
DNSWM_SERVICE_IS_ACTIVE=1
ss -utnlp|grep 53
DNS_PROC=$(systemctl is-active coredns)
if [[ "${DNS_PROC}" == "active" ]];then
    printf "\e[34mdns service is Fine! \e[0m\n"
else
    DNS_SERVICE_IS_ACTIVE=0
    printf "\e[33mWARNING!!! dns service is NOT Running! \e[0m\n"
fi

ss -utnlp|grep 9001
DNSWM_PROC=$(systemctl is-active dnswm)
if [[ "${DNSWM_PROC}" == "active" ]];then
    printf "\e[34mdnswm service is Fine! \e[0m\n"
else
    DNSWM_SERVICE_IS_ACTIVE=0
    printf "\e[33mWARNING!!! dnswm service is NOT Running! \e[0m\n"
fi

echo
if [[ ${DNS_SERVICE_IS_ACTIVE} -eq 1 ]] && [[ ${DNSWM_SERVICE_IS_ACTIVE} -eq 1 ]]; then
printf "
    install successful, then you can visit http://${DNS_SERVER_IP}:9001 to add some domain
    and check by \"nslookup *** ${DNS_SERVER_IP}\"\n\n \
    BTW, you should make sure open the port 53/udp and 9001/tcp in your firewall\n\n \
    service manage:\n \
    systemctl status|start|stop|restart dnswm\n \
    systemctl status|start|stop|restart coredns\n\n"
else
    echo "install fail"
    echo "do some clean job, please waite..."
    echo "delete \"dnswm\" user"
    userdel dnswm
    echo "clean job done"
fi
