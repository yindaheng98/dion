GO_LDFLAGS = -ldflags "-s -w"
GO_VERSION = 1.16

download:
	go mod download -x
vendor: download
	go mod tidy
	go mod vendor


go_deps: download vendor

isglb: go_deps
	go build -o isglb $(GO_LDFLAGS) github.com/yindaheng98/dion/cmd/isglb
sxu: go_deps
	go build -o sxu $(GO_LDFLAGS) github.com/yindaheng98/dion/cmd/sxu
stupid: go_deps
	go build -o stupid $(GO_LDFLAGS) github.com/yindaheng98/dion/cmd/stupid
islb: go_deps
	go build -o islb $(GO_LDFLAGS) github.com/yindaheng98/dion/cmd/islb

all: isglb sxu stupid

clean:
	rm -rf bin
	rm -rf vendor


proto: vendor protoc_install proto_core

protoc_install:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

GOPATH:=$(shell go env GOPATH)
PROTOC:=protoc
PROTOC:=$(PROTOC) --plugin=protoc-gen-go-grpc=$(GOPATH)/bin/protoc-gen-go-grpc
PROTOC:=$(PROTOC) --plugin=protoc-gen-go=$(GOPATH)/bin/protoc-gen-go
PROTOC:=$(PROTOC) --go_opt=module=github.com/yindaheng98/dion --go_out=.
PROTOC:=$(PROTOC) --go-grpc_opt=module=github.com/yindaheng98/dion --go-grpc_out=.
PROTOC:=$(PROTOC) -I ./vendor -I ./

proto_core:
	$(PROTOC) proto/isglb.proto
	$(PROTOC) proto/room.proto
