# near_exporter

Docker images are available on [Docker Hub](https://hub.docker.com/r/certusone/near_exporter).

Go is required to build the binary.

***Build the binary:***
```
go build github.com/certusone/near_exporter/cmd/near_exporter
```

***Systemd service example***
```
cp near_exporter /usr/local/bin

vi /etc/systemd/system/near-exporter.service

#Add this content to the file:

[Unit]
Description=near-exporter
After=network.target

[Service]
Environment="NEAR_RPC_ADDR=http://127.0.0.1:3030"
ExecStart=/usr/local/bin/near_exporter
Restart=always
RestartSec=10s

[Install]
WantedBy=multi-user.target


systemctl enable --now near-exporter
```

Exporter will be available at http://localhost:8080/metrics
