FROM golang:1.11.1-alpine as builder

RUN apk add --update --no-cache git gcc musl-dev linux-headers make

ENV APP_NAME="pm-kyc-service"
# Set up the environment to use the workspace.
ENV APP_DIR=/go/src/github.com/gnosis/${APP_NAME}
RUN mkdir -p $APP_DIR
ENV GOPATH="/go"

# dep as dependency manager
RUN go get -u github.com/golang/dep/cmd/dep

# bee command line, for generating docs and compiling
RUN go get github.com/beego/bee

COPY Gopkg.toml Gopkg.lock $APP_DIR/

# Install dependencies
RUN cd ${APP_DIR} && dep ensure -v -vendor-only

# Easy fix to https://github.com/golang/dep/issues/1847
RUN go get github.com/ethereum/go-ethereum
RUN cp -r \
  "${GOPATH}/src/github.com/ethereum/go-ethereum/crypto/secp256k1/libsecp256k1" \
  "${APP_DIR}/vendor/github.com/ethereum/go-ethereum/crypto/secp256k1/"

ADD . ${APP_DIR}
RUN go get

# Compile files
RUN cd ${APP_DIR} && $GOPATH/bin/bee generate docs && ONLY_COMPILE=true go run main.go && go build

# Pull Geth into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates tini
COPY --from=builder /go/src/github.com/gnosis/pm-kyc-service/pm-kyc-service /usr/local/bin/
WORKDIR /root
COPY --from=builder /go/src/github.com/gnosis/pm-kyc-service/swagger ./swagger
COPY --from=builder /go/src/github.com/gnosis/pm-kyc-service/prod-conf ./conf

ENTRYPOINT ["/sbin/tini", "--"]

EXPOSE 8080
