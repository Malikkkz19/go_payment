.PHONY: proto
proto:
	protoc --go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		api/proto/payment/v1/payment.proto

.PHONY: build
build:
	go build -o bin/payment-api ./cmd/api
	go build -o bin/payment-grpc ./cmd/grpc

.PHONY: run-api
run-api:
	go run ./cmd/api

.PHONY: run-grpc
run-grpc:
	go run ./cmd/grpc

.PHONY: test
test:
	go test -v ./...

.PHONY: docker-build
docker-build:
	docker build -t payment-service .

.PHONY: docker-compose-up
docker-compose-up:
	docker-compose up -d

.PHONY: docker-compose-down
docker-compose-down:
	docker-compose down
