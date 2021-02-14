# near_exporter

Docker images are available on [Docker Hub](https://hub.docker.com/r/certusone/near_exporter).

Go >= 1.14 is required to build the binary.

### How to build

```
go build github.com/certusone/near_exporter/cmd/near_exporter
```

### systemd service example

```
cp near_exporter /usr/local/bin

cat <<EOF > /etc/systemd/system/near-exporter.service
[Unit]
Description=Certus One near_exporter
Documentation=https://github.com/certusone/near_exporter
After=network.target

[Service]
Environment="NEAR_RPC_ADDR=http://127.0.0.1:3030"
ExecStart=/usr/local/bin/near_exporter
Restart=always
RestartSec=10s

[Install]
WantedBy=multi-user.target
EOF
```

Enable and start the service:

```
systemctl enable --now near-exporter
```

Exporter will be available at http://localhost:8080/metrics
