export DATA_PATH=/opt/dnswm
export DNSWM_PORT=9001
mkdir -p ${DATA_PATH}/zones
mkdir -p ${DATA_PATH}/bin
curl -L https://github.com/simanchou/dnswm/releases/download/v1.0/dnswm-linux-1.0.tar.gz -o ${DATA_PATH}/dnswm-linux-1.0.tar.gz
cd ${DATA_PATH}
tar zxf dnswm-linux-1.0.tar.gz
chmod +x dnswm
chmod +x coredns
mv coredns ${DATA_PATH}/bin/coredns
mv dnswm ${DATA_PATH}/bin/dnswm
mv assets ${DATA_PATH}/bin/assets
mv tmpl ${DATA_PATH}/bin/tmpl
export DNS_SERVER_IP=$(hostname -I | cut -d" " -f 1)
useradd -M -s /sbin/nologin dnswm

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


systemctl daemon-reload
systemctl restart coredns
systemctl restart dnswm

