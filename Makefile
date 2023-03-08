.PHONY: build test testw sqlite vhs

build:
	go build -o nom cmd/nom/main.go

test:
	go test -v ./internal/...

testw:
	gotestsum --watch

sqlite:
	docker run --rm -it -v "${HOME}/.config/nom:/workspace" keinos/sqlite3 sqlite3 /workspace/nom.db -header -column

vhs:
	docker run --rm -v $PWD:/vhs ghcr.io/charmbracelet/vhs .github/demo.tape
