.PHONY: gen-proto install-tools test

# Generate Go protobuf and gRPC stubs
gen-proto:
	@echo "Generating protobuf files..."
	protoc --proto_path=proto --go_out=paths=source_relative:pb --go-grpc_out=paths=source_relative:pb proto/common/common.proto proto/inventory/inventory_service.proto

# Install protoc plugins for Go
install-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Run unit tests across all grpc packages
test:
	go test -v ./grpc/...
