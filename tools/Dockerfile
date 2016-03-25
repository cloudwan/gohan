FROM karalabe/xgo-latest

# Docker file for cross compiling gohan

ENTRYPOINT ["/bin/bash"]

ENV GOPATH /go
ENV PATH  $GOPATH/bin:$PATH

ADD dev_setup.sh /dev_setup.sh
RUN chmod +x /dev_setup.sh
RUN /dev_setup.sh

MAINTAINER Nachi Ueno <nati.ueno@gmail.com>