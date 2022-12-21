build:
	go build -o nom cmd/nom/main.go

test:
	go test -v ./internal/...
