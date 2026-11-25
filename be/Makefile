.PHONY: run-gateway run-user run-event run-inventory run-order run-all

run-gateway:
	cd gateway && go run cmd/main.go

run-user:
	cd user-service && go run cmd/server/main.go

run-event:
	cd event-service && go run cmd/server/main.go

run-inventory:
	cd inventory-service && go run server/main.go

run-order:
	cd order-service && go run server/main.go

run-all: run-order run-inventory run-event run-user run-gateway