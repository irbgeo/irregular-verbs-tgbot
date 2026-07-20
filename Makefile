.PHONY: lint

lint:
	golangci-lint run --fix

.PHONY: mocks

# Regenerate mockery mocks (config in .mockery.yml).
mocks:
	mockery
