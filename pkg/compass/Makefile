# dep:
# 	go get github.com/ghodss/yaml
# 	go get github.com/gogo/protobuf/proto
# 	go get github.com/gogo/protobuf/types
# 	go get google.golang.org/grpc
# 	go get github.com/envoyproxy/go-control-plane/envoy/api/v2
# 	go get github.com/envoyproxy/go-control-plane/envoy/api/v2/core
# 	go get github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint
# 	go get github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2

common: common/*.go
	go install ./common

envoy: routers/envoy/*.go
	go install ./routers/envoy

routers: routers/*.go
	go install ./routers

admin: admin/*.go
	go install ./admin

main: *.go
	go build

build: common envoy router admin main

all: build

# release:
# 	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_LINUX) -v

