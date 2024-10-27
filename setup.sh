#!/bin/bash

# Update package list and install Chrony
echo "Updating system and installing Chrony..."
sudo apt update
sudo apt install chrony golang libpcap-dev tmux -y

# Configure Chrony server
echo "Configuring Chrony server..."
sudo sed -i 's/^pool/#pool/g' /etc/chrony/chrony.conf  # Comment out default pool servers
echo "
server 132.239.1.6 iburst

# Custom NTP servers
server 0.pool.ntp.org iburst
server 1.pool.ntp.org iburst

# Allow local network clients (replace with your actual network IP range)
allow 0.0.0.0/0
" | sudo tee -a /etc/chrony/chrony.conf > /dev/null

# Start and enable Chrony service
echo "Starting and enabling Chrony service..."
sudo systemctl restart chrony
sudo systemctl enable chrony

tmux new -s ntp_logger