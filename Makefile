MAKEFLAGS += --silent

PROTO_SERVICE_PATH = $(shell pwd)/apis/userservice/v1
PROTO_INCLUDE_PATH = -I=. -I/usr/include -I$(PROTO_SERVICE_PATH)

.PHONY: build_proto
build_proto:
	protoc $(PROTO_INCLUDE_PATH) --go_out=internal/userservice --grpc-go_out=internal/userservice $(PROTO_SERVICE_PATH)/userservice.proto
	protoc $(PROTO_INCLUDE_PATH) --grpc-gateway_out=logtostderr=true:internal/userservice $(PROTO_SERVICE_PATH)/userservice.proto
	protoc $(PROTO_INCLUDE_PATH) --swagger_out=logtostderr=true:api/swagger/v1 $(PROTO_SERVICE_PATH)/userservice.proto

.PHONY: build
build:
	CGO_ENABLED=0 go build -race -o $(PWD)/bin/userservice

.PHONY: run
run: build
	./bin/hometown --config config/dev.yaml start
