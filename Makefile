BUF_VERSION:=1.4.0

install:
	go install github.com/bufbuild/buf/cmd/buf@v$(BUF_VERSION)

format:
	@buf format -w

generate:
	@buf generate

lint:
	@buf lint
	@buf breaking --against 'https://github.com/johanbrandhorst/connect-gateway-example.git#branch=master'

.PHONY: install format generate lint
