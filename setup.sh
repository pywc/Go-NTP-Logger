#!/bin/sh

sudo apt update
sudo apt install libpcap-dev tmux golang -y
sudo ufw allow 123/udp
go get
tmux new -s ntp_server
