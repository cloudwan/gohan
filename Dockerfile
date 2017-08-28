FROM golang:1.9
MAINTAINER Leif Madsen <leif@leifmadsen.com>

RUN go get github.com/cloudwan/gohan

ENTRYPOINT ["/go/bin/gohan"]
CMD ["server", "--config-file", "/go/src/github.com/cloudwan/gohan/etc/gohan.yaml"]
