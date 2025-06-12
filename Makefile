# golangci-lint
LINTER := bin/golangci-lint
LINTER_CONFIG=.golangci.yml

$(LINTER):
	curl -SL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s latest


.PHONY: fix
# lint fix
fix: $(LINTER)
	@$(LINTER) run -v --fix --timeout=5m --config=$(LINTER_CONFIG)
	@echo "lint fix finished"

.PHONY: lint
# lint check
lint: $(LINTER)
	@$(LINTER) run --timeout=5m --config=$(LINTER_CONFIG)
	@echo "lint check finished"

# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help