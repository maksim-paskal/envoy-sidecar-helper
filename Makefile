KUBECONFIG=$(HOME)/.kube/dev
tag=dev

lint:
	ct lint --all
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
	docker build --pull --push . -t paskalmaksim/envoy-sidecar-helper:$(tag)
deploy:
	rm -rf ./examples/envoy-sidecar-helper-test/charts
	helm dep up ./examples/envoy-sidecar-helper-test --skip-refresh

	helm upgrade --install envoy-sidecar-helper-test \
	--namespace envoy-sidecar-helper \
	--create-namespace \
	--set envoy-sidecar-helper.image.tag=$(tag) \
	--set envoy-sidecar-helper.pullPolicy=Always \
	./examples/envoy-sidecar-helper-test
clean:
	helm -n envoy-sidecar-helper delete envoy-sidecar-helper-test || true
	kubectl delete ns envoy-sidecar-helper