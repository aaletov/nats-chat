FROM golang:alpine3.18
WORKDIR /opt
RUN apk update && \
  apk add python3 py3-pip git
ADD https://github.com/nats-io/nats-server/releases/download/v2.9.20/nats-server-v2.9.20-linux-amd64.zip nats-server.zip
RUN unzip nats-server.zip -d nats-server && \
  cp nats-server/nats-server-v2.9.20-linux-amd64/nats-server /usr/bin
COPY --from=nats:alpine3.18 /etc/nats/nats-server.conf /etc/nats/nats-server.conf
COPY ./go.* .
RUN go mod download
COPY . ./nats-chat
WORKDIR /opt/nats-chat
ENV NATS_CHAT_HOME=/opt/nats-chat
RUN mkdir -p ./build && go build -o ./build/nats-chat
CMD ["/bin/sh", "./test/test.sh"]