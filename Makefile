.PHONY: build test all clean

install-proto:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

generate-proto:
	mkdir -p ./api/generated 
	protoc --proto_path=./api \
		--go_out=./api/generated \
		--go_opt=paths=source_relative \
		--experimental_allow_proto3_optional=true \
		./api/api.proto

build:
	mkdir -p ./build
	go build -o ./build/nats-chat

test:
	docker build . -t nats-chat-test:latest
	docker run --rm nats-chat-test:latest