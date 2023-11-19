LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: gen_docs
gen_docs: ## Generate helm documentation
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install github.com/norwoodj/helm-docs/cmd/helm-docs@latest
	./bin/helm-docs --template-files=./README.md.gotmpl --sort-values-order file
