.PHONY: proto buf-lint

# Regenerate Go code from proto definitions (remote buf plugins, no local protoc needed)
proto:
	buf generate

# Lint proto files
buf-lint:
	buf lint
