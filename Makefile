TAG := $(shell git rev-parse --short HEAD)

install-proto:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

generate-proto:
	mkdir -p ./api/generated 
	protoc --proto_path=./api \
		--go_out=./api/generated \
		--go_opt=paths=source_relative \
		--go-grpc_out=./api/generated \
		--go-grpc_opt=paths=source_relative \
		--experimental_allow_proto3_optional=true \
		./api/api.proto

build-cli:
	mkdir -p ./build
	go build -o ./build/nats-chat-cli ./cmd/client/main.go

build-daemon:
	mkdir -p ./build
	go build -o ./build/nats-chat-daemon ./cmd/server/main.go

build-all: build-cli build-daemon

docker-build-builder:
	docker build -f Dockerfile.builder -t nats-chat-builder:$(TAG) .
	docker tag nats-chat-builder:$(TAG) nats-chat-builder:latest

docker-build-cli: docker-build-builder
	docker build -f Dockerfile.cli -t nats-chat-cli:$(TAG) .
	docker tag nats-chat-cli:$(TAG) nats-chat-cli:latest

docker-build-daemon: docker-build-builder
	docker build -f Dockerfile.daemon -t nats-chat-daemon:$(TAG) .
	docker tag nats-chat-daemon:$(TAG) nats-chat-daemon:latest

docker-build-all: docker-build-cli docker-build-daemon

.PHONY: test
test: docker-build-all
	python3 ./test/test.py

clean:
	rm -rf build/*