LOCAL_BIN:=$(CURDIR)/bin

.PHONY: test
test:
	@echo "\n --- Run project tests --- \n"
	go test ./...

.PHONY: bin-deps
bin-deps:
	GOBIN=$(LOCAL_BIN) go install github.com/pav5000/smartimports/cmd/smartimports@v0.2.0

.PHONY: format
format:
	@echo "\n --- Start format imports --- \n"
	$(LOCAL_BIN)/smartimports -local "github.com/themgmd/scenario"
