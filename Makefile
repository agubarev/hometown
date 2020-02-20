.PHONY: build run proto clean deploy
MAKEFLAGS += --silent

build: 
	CGO_ENABLED=0 go build -race -o $(PWD)/bin/hometown

run: build
	./bin/hometown --config config/dev.yaml start
