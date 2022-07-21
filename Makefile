KUBECONFIG=$(HOME)/.kube/dev

test:
	./scripts/validate-license.sh
	go mod tidy
	go fmt ./cmd/... ./pkg/...
	go vet ./cmd/... ./pkg/...
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run -v
run:
	go run -race ./cmd \
	-log.pretty \
	-kubeconfig=$(KUBECONFIG)
build:
	go run github.com/goreleaser/goreleaser@latest build --rm-dist --snapshot
	mv ./dist/envoy-sidecar-helper_linux_amd64_v1/envoy-sidecar-helper envoy-sidecar-helper
	docker build --pull --push . -t paskalmaksim/envoy-sidecar-helper:dev
deploy:
	make clean || true
	kubectl apply -f ./examples
clean:
	kubectl delete -f ./examples