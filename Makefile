MAKEFLAGS += --silent

PROTO_SERVICE_PATH = $(shell pwd)/api
PROTO_INCLUDE_PATH = -I=. -I/usr/include \
						-I$(GOPATH) \
						-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
						-I$(PROTO_SERVICE_PATH)

.PHONY: grpc_deps
grpc_deps:
	go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
	go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-openapiv2
	go get -u github.com/golang/protobuf/protoc-gen-go

.PHONY: build_proto
build_proto:
	protoc $(PROTO_INCLUDE_PATH) \
		--go_out=internal/userservice/proto \
		--go-grpc_out=internal/userservice/proto \
		--grpc-gateway_out=logtostderr=true:internal/userservice/proto \
		--openapiv2_out=use_go_templates=true:$(GOPATH) \
		$(PROTO_SERVICE_PATH)/userservice/v1/userservice.proto

.PHONY: build
build:
	CGO_ENABLED=0 go build -race -o $(PWD)/bin/userservice

.PHONY: run
run: build
	./bin/hometown --config config/dev.yaml start
