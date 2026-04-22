BINARY=ai-council

build:
	go build -o $(BINARY) .

install: build
	cp $(BINARY) $(shell go env GOPATH)/bin/$(BINARY)

.PHONY: build install
