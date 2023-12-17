#!/usr/bin/env bash

GOBIN="/usr/local/bin" sudo go install github.com/appkins-org/go-maaspower@v0.0.2

sudo mkdir -p /etc/maaspower && cat <<EOF | sudo tee /etc/maaspower/config.yaml
username: "$UNIFI_USERNAME"
password: "$UNIFI_PASSWORD"
api_endpoint: "https://10.0.0.1:443"
EOF

cat <<EOF | sudo tee /etc/systemd/system/maaspower.service
[Unit]
Description=maaspower daemon
[Service]
ExecStart=/usr/local/bin/maaspower -c=/etc/maaspower/config.yaml
[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable maaspower.service
sudo systemctl start maaspower.service
sudo systemctl status maaspower.service