CONTROLLER_GEN := go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.22.0

.PHONY: generate install uninstall run test

generate:
	$(CONTROLLER_GEN) object paths=./api/...
	$(CONTROLLER_GEN) crd paths=./api/... output:crd:artifacts:config=config/crd

install: generate
	kubectl apply -f config/crd

uninstall:
	kubectl delete -f config/crd

run:
	go run ./cmd

test:
	go test ./...
