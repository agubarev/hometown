.PHONY: build run proto
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

build: proto 
	go install

run: proto
	go run main.go