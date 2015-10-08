#!/bin/bash

# Cross Compile helper code for gohan
# you need ubuntu14.04 + docker env
#
# How to setup cross comile env
#
# docker pull karalabe/xgo-latest
# docker run --entrypoint="/bin/bash" -v /vagrant:/build -i -t karalabe/xgo-latest
#
# inside docker
#
# export PATH=$PATH:/go/bin
# export GOPATH=/go
# go get github.com/tools/godep
# go get github.com/golang/lint/golint
# go get github.com/coreos/etcd
# go get golang.org/x/tools/cmd/cover
# export GOHAN=$GOPATH/src/github.com/cloudwan/gohan
# mkdir -p $GOPATH/src/github.com/cloudwan/
# cd $GOPATH/src/github.com/cloudwan/
# git clone https://nati_ueno@github.com/cloudwan/gohan.git
# cd gohan
# make
# make install

rm -rf build
mkdir build
DIR=`pwd`

./download_webui.sh

cd $DIR/docs
make singlehtml

rm -rf /build/gohan*
/build.sh github.com/cloudwan/gohan/gohan

for dist in linux-386 linux-amd64 linux-arm darwin-386 darwin-amd64 windows-386.exe windows-amd64.exe
  do
      echo $dist
      DIST_DIR=$DIR/build/gohan-$dist/
      mkdir $DIST_DIR
      git log -n 5 > $DIST_DIR/git-commit.log
      cp /build/gohan-$dist $DIST_DIR/gohan
      cp -r $DIR/etc $DIST_DIR
      cp -r $DIR/docs/build/singlehtml $DIST_DIR/docs
      cp -r $DIR/docs/build/singlehtml $DIST_DIR/etc/webui/docs

      cd $DIR/build
      zip -r gohan-$dist.zip gohan-$dist
      rm -rf $DIST_DIR
  done
