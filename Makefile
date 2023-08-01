.PHONY: build test all clean

build:
	mkdir -p ./build
	go build -o ./build/nats-chat

test: build
	docker run --rm -d --name nats -p 4444:4444 nats:alpine3.18 -p 4444 -D --trace
	python3 ./test/test-hello.py || docker stop nats 
	docker stop nats