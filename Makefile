run:
	@echo "Running WAL-go"
	go run .

test:
	@echo "Running WAL-go tests"
	go test .

.PHONY: run test