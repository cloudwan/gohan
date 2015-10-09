FROM ubuntu:14.04

MAINTAINER Karthik Ramasubramanian <karthik@ntti3.com>

ENV GO_VERSION 1.4.2
ENV GOPATH  /go
ENV GOHAN_PATH $GOPATH/src/github.com/cloudwan/gohan

# Install go
ADD https://storage.googleapis.com/golang/go$GO_VERSION.linux-amd64.tar.gz /
RUN tar xzvf go$GO_VERSION.linux-amd64.tar.gz
RUN mv /go /usr/local/go

# Update apt
RUN apt-get update

# Install dependencies for go-get
RUN apt-get install -y bzr \
                       curl \
                       git \
                       mercurial

# Install build tools
RUN apt-get install -y build-essential

# Setup environment varialbles for go
# NOTE: It is now recommended to not set GOROOT.
ENV PATH  /usr/local/go/bin:/go/bin:$PATH

# Bundle app source
ADD . /src

WORKDIR $GOHAN_PATH
ADD . $GOHAN_PATH
# Jenkins hack that adds keys.
RUN rm -rf keys

# Install first few dependencies
RUN go get github.com/tools/godep
RUN go get github.com/golang/lint/golint
RUN go get github.com/coreos/etcd
RUN go get golang.org/x/tools/cmd/cover
RUN go get golang.org/x/tools/cmd/vet

# Build gohan
RUN cd $GOHAN_PATH; make all install

ENTRYPOINT ["gohan", "server", "--config-file", "etc/gohan.yaml"]
