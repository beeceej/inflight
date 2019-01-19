.PHONY: get-tools
get-tools:
	@echo "+ $@: retrieving build/test dependencies "
	@go get -u -v github.com/google/go-cmp/...

test:
	@echo 'run `make get-tools` if you are unable to run this test`'
	@go test ./... -race -coverprofile=coverage.txt -covermode=atomic