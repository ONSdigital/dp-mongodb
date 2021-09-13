test:
	go test -race -cover ./...
.PHONY: test

test-component:
.PHONY: test-component

audit:
	go list -json -m all | nancy sleuth
.PHONY: audit

build:
	go build ./...
.PHONY: build

lint:
	go fmt ./...
.PHONY: lint
