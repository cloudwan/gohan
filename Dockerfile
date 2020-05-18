ARG GO_VERSION=1.13


FROM golang:$GO_VERSION-alpine
RUN apk add --no-cache gcc libc-dev

WORKDIR $GOPATH/src/github.com/cloudwan/gohan

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go install -v

ENTRYPOINT ["/go/bin/gohan"]
