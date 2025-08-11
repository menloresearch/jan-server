.PHONY: install
install:
	@go install github.com/swaggo/swag/cmd/swag@latest
	@go install github.com/google/wire/cmd/wire@latest

# .PHONY: doc
# doc:
# 	@swag init --parseDependency -g server.go -d application/server/http -o docs

.PHONY: wire
wire:
	@wire ./cmd/server

.PHONY: setup
setup:
	@make wire