.PHONY: build test all clean

build:
	mkdir -p ./build
	go build -o ./build/nats-chat

test: build
	python3 ./test/test-hello.py