#!/bin/sh

sudo apt update
sudo apt install tmux golang -y
sudo ufw allow 123/udp
go get
tmux new -s reddit
