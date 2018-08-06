# dep:
# 	go get github.com/ghodss/yaml
# 	go get github.com/gogo/protobuf/proto
# 	go get github.com/gogo/protobuf/types
# 	go get google.golang.org/grpc
# 	go get github.com/envoyproxy/go-control-plane/envoy/api/v2
# 	go get github.com/envoyproxy/go-control-plane/envoy/api/v2/core
# 	go get github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint
# 	go get github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2
#		go get github.com/go-sql-driver/mysql
#  	go get github.com/sirupsen/logrus
#		go get github.com/gorilla/mux


all: build

.PHONY: common
common: common/*.go
	go install ./common

.PHONY: mysql
mysql: stores/mysql/*.go
	go install ./stores/mysql

.PHONY: stores
stores: stores/*.go
	go install ./stores

.PHONY: envoy
envoy: routers/envoy/*.go
	go install ./routers/envoy

.PHONY: routers
routers: routers/*.go
	go install ./routers

.PHONY: admin
admin: admin/*.go
	go install ./admin

.PHONY: main
main: *.go
	go build

.PHONY: build
build: common mysql stores envoy routers admin main

# release:
# 	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_LINUX) -v

