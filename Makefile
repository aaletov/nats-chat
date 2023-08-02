.PHONY: build test all clean

build:
	mkdir -p ./build
	go build -o ./build/nats-chat

test:
	docker build . -t nats-chat-test:latest
	docker run --rm nats-chat-test:latest