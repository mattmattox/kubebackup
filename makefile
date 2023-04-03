.PHONY: install build test run docker-build

install:
	go get -d -v ./...

build:
	go get -d -v ./...
	go build -v -o kubebackup

test:
	go test -v ./...

run:
	go run main.go

docker-build:
	docker build -t cube8021/kubebackup:latest .

compile:
	GOOS=linux GOARCH=amd64 go build -o bin/kubebackup-linux-amd64 main.go
	GOOS=linux GOARCH=arm64 go build -o bin/kubebackup-linux-arm64 main.go
