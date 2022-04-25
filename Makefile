GO_LDFLAGS = -ldflags "-s -w"
GO_VERSION = 1.16

all: go_deps core

go_deps:
	go mod download -x && go mod vendor

core:
	go build -o bin/islb $(GO_LDFLAGS) cmd/isglb/main.go

clean:
	rm -rf bin
	rm -rf vendor

vendor:
	go mod vendor

proto: vendor protoc_install proto_core

protoc_install:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

GOPATH:=$(shell go env GOPATH)
PROTOC:=protoc
PROTOC:=$(PROTOC) --plugin=protoc-gen-go-grpc=$(GOPATH)/bin/protoc-gen-go-grpc
PROTOC:=$(PROTOC) --plugin=protoc-gen-go=$(GOPATH)/bin/protoc-gen-go
PROTOC:=$(PROTOC) --go_opt=module=github.com/yindaheng98/isglb --go_out=.
PROTOC:=$(PROTOC) --go-grpc_opt=module=github.com/yindaheng98/isglb --go-grpc_out=.
PROTOC:=$(PROTOC) -I ./vendor -I ./

proto_core:
	$(PROTOC) proto/isglb.proto
