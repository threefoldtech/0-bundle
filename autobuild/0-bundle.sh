#!/bin/bash
set -ex

ncpu=$(grep 'model name' /proc/cpuinfo | wc -l)

apt-get update
apt-get install -y curl git build-essential
apt-get install -y libgflags-dev # libzstd-dev

# install go 1.8 (needed by fuse)
curl https://storage.googleapis.com/golang/go1.10.1.linux-amd64.tar.gz > /tmp/go1.10.1.linux-amd64.tar.gz
tar -C /usr/local -xzf /tmp/go1.10.1.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
mkdir -p /gopath
export GOPATH=/gopath

# 0-bundle
mkdir -p $GOPATH/src/github.com/threefoldtech/0-bundle
cp -ar /0-bundle $GOPATH/src/github.com/threefoldtech/

pushd $GOPATH/src/github.com/threefoldtech/0-bundle

make build

# reduce binary size
strip -s zbundle

# print shared libs
ldd zbundle || true
popd

mkdir -p /tmp/root/bin
cp $GOPATH/src/github.com/threefoldtech/0-bundle/zbundle /tmp/root/bin

mkdir -p /tmp/archives/
tar -czf "/tmp/archives/0-bundle.tar.gz" -C /tmp/root .
