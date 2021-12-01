test:
	go test -race -cover ./...
.PHONY: test

test-component:
	go test -race -cover -v ./testcomponent -component
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
