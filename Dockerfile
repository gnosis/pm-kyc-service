FROM golang:1.10-alpine as builder

RUN apk add --update --no-cache tini git

ENV APP_NAME="pm-kyc-service"
# Set up the environment to use the workspace.
ENV APP_DIR=/go/src/github.com/gnosis/${APP_NAME}
RUN mkdir -p $APP_DIR
ENV GOPATH="/go"

# dep as dependency manager
RUN go get -u github.com/golang/dep/cmd/dep

# bee command line, for generating docs and compiling
RUN go get github.com/beego/bee

ADD . ${APP_DIR}

# Install dependencies
RUN cd ${APP_DIR} && dep ensure -v

# Compile files
# RUN go build

# ENTRYPOINT ["$GOPATH/bin/bee run -downdoc=true -gendoc=true"]
ENTRYPOINT ["/sbin/tini", "--"]

EXPOSE 8080
