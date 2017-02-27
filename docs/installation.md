# Installation

You can download Gohan binary for your platfrom from
github release page.


## Download Binary

* Download [Gohan Release](https://github.com/cloudwan/gohan/releases)
* Start server: `./gohan server --config-file etc/gohan.yaml`

## Build

* Install GO >= 1.6
* go get github.com/cloudwan/gohan

# Packages

## Ubuntu 14.04 Trusty 64bits server
```bash
wget -qO - https://deb.packager.io/key | sudo apt-key add -
echo "deb https://deb.packager.io/gh/cloudwan/gohan trusty master" | sudo tee /etc/apt/sources.list.d/gohan.list
sudo apt-get update
sudo apt-get install gohan
```
## CentOS / RHEL 6 64 bits server

```bash
sudo rpm --import https://rpm.packager.io/key
echo "[gohan]
name=Repository for cloudwan/gohan application.
baseurl=https://rpm.packager.io/gh/cloudwan/gohan/centos6/master
enabled=1" | sudo tee /etc/yum.repos.d/gohan.repo
sudo yum install gohan
```

## Debian 7 Wheezy 64bits server

```bash
wget -qO - https://deb.packager.io/key | sudo apt-key add -
echo "deb https://deb.packager.io/gh/cloudwan/gohan wheezy master" | sudo tee /etc/apt/sources.list.d/gohan.list
sudo apt-get update
sudo apt-get install gohan
```

## Heroku

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy?template=https://github.com/cloudwan/gohan.git)
