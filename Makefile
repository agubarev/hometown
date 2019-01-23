.PHONY: build run proto clean deploy
MAKEFLAGS += --silent

build: 
	go build -o $(PWD)/bin/hometown

run: build
	./bin/hometown --config config/dev.yaml start