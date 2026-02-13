.PHONY: build test clean run-coordinator run-node

VERSION ?= dev

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o bin/claw-mesh ./cmd/claw-mesh

test:
	go test -race -count=1 ./...

clean:
	rm -rf bin/ dist/

run-coordinator: build
	./bin/claw-mesh up --port 9180

run-node: build
	./bin/claw-mesh join http://127.0.0.1:9180 --name local-node

lint:
	golangci-lint run ./...

release-dry:
	goreleaser release --snapshot --clean

release:
	goreleaser release --clean
