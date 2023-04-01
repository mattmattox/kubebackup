.PHONY: build test docker-build

build:
	go build -v -o kubebackup

test:
	go test -v ./...

docker-build:
	docker build -t cube8021/kubebackup:latest .
