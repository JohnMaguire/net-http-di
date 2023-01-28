bin:
	go build $(BUILD_FLAGS) -o ./server

fmt:
	goimports -w ./..

test:
	go test ./...

vet:
	go vet ./...

.PHONY: bin fmt test vet
