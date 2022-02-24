build:
	go build -o bin/k8cli

.PHONY: run
run:
	go run *.go

.PHONY: rungcp
rungcp:
	go run *.go --operation cluster --context gcp-cluster1      

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

