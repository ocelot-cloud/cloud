#!/bin/bash

if [ "$(id -u)" = "0" ]; then
    echo "This script should not be run as root" 1>&2
    exit 1
fi

echo "Installing required debian packages"
sudo apt-get update
# basic tools
sudo apt-get install -y wget git curl sshpass nmap
# used for building the backend golang executable for the alpine Docker image
sudo apt-get musl-tools
# The libraries needed by cypress: https://docs.cypress.io/guides/getting-started/installing-cypress#UbuntuDebian
sudo apt-get install -y libgtk2.0-0 libgtk-3-0 libgbm-dev libnotify-dev libnss3 libxss1 libasound2

echo "Installing docker"
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER

echo "Installing go"
wget https://go.dev/dl/go1.22.8.linux-amd64.tar.gz -O go.tar.gz
sudo tar -C /usr/local -xzf go.tar.gz
rm go.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> $HOME/.bashrc
mkdir -p "$HOME/.go-cache"
echo "export GOPATH=$HOME/.go-cache" >> $HOME/.bashrc

echo "Installing node.js"
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash
source $HOME/.nvm/nvm.sh
nvm install 22.11
nvm use 22.11
echo 'export PATH=$PATH:/usr/local/lib/node_modules/.bin' >> $HOME/.bashrc

source $HOME/.bashrc
go install mvdan.cc/garble@latest
echo "Please reboot the system manually to complete the installation."