.PHONY: gifs

all: gifs

VERSION=v0.0.6

TAPES=$(shell ls doc/vhs/*tape)
gifs: $(TAPES)
	for i in $(TAPES); do vhs < $$i; done

docker-lint:
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:v1.50.1 golangci-lint run -v

lint:
	golangci-lint run -v --enable=exhaustive

test:
	go test ./...

build:
	go generate ./...
	go build ./...

tag-major:
	git tag $(shell svu major)

tag-minor:
	git tag $(shell svu minor)

tag-patch:
	git tag $(shell svu patch)

goreleaser:
	goreleaser release --skip-sign --snapshot --rm-dist

release:
	git push --tags
	GOPROXY=proxy.golang.org go list -m github.com/go-go-golems/clay@$(shell svu current)

exhaustive:
	golangci-lint run -v --enable=exhaustive

bump-glazed:
	go get -v -t -u github.com/go-go-golems/glazed@latest
	go mod tidy

CLAY_BINARY=$(shell which clay)

install:
	go build -o ./dist/clay ./cmd/clay && \
		cp ./dist/clay $(CLAY_BINARY)
