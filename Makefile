

checks:
	@go fmt ./...
	@staticcheck ./...
	@gosec ./...
	@goimports -w .
