FROM golang:1.20-alpine3.18
WORKDIR /opt
RUN apk update && \
  apk add git make
COPY ./go.* .
RUN go mod download
