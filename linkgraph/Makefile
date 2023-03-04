include ../MakeFile

proto: check-proto-deps
	@echo "[protoc].....generating protos for API....."
	@protoc --go_out=./store/api/rpc/proto/ --go-grpc_out=./store/api/rpc/proto/ -I github.com/mycok/uSearch/linkgraph/store/api/rpc/proto api.proto

check-proto-deps:
	@echo "[go get] .....checking if protoc packages are available....."
	@go get google.golang.org/grpc
	@go get github.com/golang/protobuf/protoc-gen-go

mock-gen: check-mockgen-deps
	@echo "[mockgen].....generating client mocks....."
	@mockgen -package mocks -destination ./store/api/rpc/mocks/mocks.go github.com/mycok/uSearch/linkgraph/store/api/rpc/proto LinkGraphClient,LinkGraph_LinksClient

check-mockgen-deps:
	@echo "[go get] .....checking if the mockgen package are available....."
	@go get github.com/golang/mock