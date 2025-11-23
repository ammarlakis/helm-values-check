# Tasks for helm-values-check

default: build

build:
	go build ./...

run chart_path:
	go run ./cmd/helm-values-check {{chart_path}}

test:
	go test ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy

release version:
	./scripts/release.sh {{version}}
