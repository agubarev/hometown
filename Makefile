.PHONY: build run proto clean deploy
MAKEFLAGS += --silent

proto:
	protoc -I/usr/local/include -I. -I$(GOPATH)/src \
		-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		-I$(GOPATH)/src/github.com/gogo/protobuf/protobuf \
		--go_out=plugins=grpc:$(GOPATH)/src gitlab.com/agubarev/hometown/rpc/user/userservice.proto

	protoc -I/usr/local/include -I. -I$(GOPATH)/src \
		-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--grpc-gateway_out=logtostderr=true:$(GOPATH)/src \
		gitlab.com/agubarev/hometown/rpc/user/userservice.proto

build: 
	go build -o $(PWD)/bin/hometown

run: build
	./bin/hometown --config config/config.yaml