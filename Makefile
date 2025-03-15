all: fmt lint

tools:
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin ${LINT_VERSION}

fmt:
	@golangci-lint run -c .golangci.yml --disable-all --enable=gci --fix

lint:
	@golangci-lint --version
	@golangci-lint run -c .golangci.yml
