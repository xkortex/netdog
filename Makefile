VERSION := $(shell git describe --always --dirty --tags)

.PHONY: default get test all vet

default:
	go build -i -ldflags="-X 'main.Version=${VERSION}'" -o ${GOPATH}/bin/netdog


all: fmt get vet default


get:
	go get

fmt:
	go fmt ./...

static: get
	CGO_ENABLED=0 go build -i -ldflags="-X 'main.Version=${VERSION}'" -o ${GOPATH}/bin/netdog

vet:
	go vet ./...

