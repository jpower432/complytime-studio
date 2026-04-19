# SPDX-License-Identifier: Apache-2.0

PORT ?= 8080
KIND_CLUSTER ?= complytime-studio
NAMESPACE ?= kagent
GATEWAY_IMAGE ?= studio-gateway
GATEWAY_TAG ?= local

CLICKHOUSE ?= true

HELM_AUTH_FLAGS :=
ifdef GITHUB_CLIENT_ID
HELM_AUTH_FLAGS += --set auth.github.clientId=$(GITHUB_CLIENT_ID)
endif
ifdef VERTEX_PROJECT_ID
HELM_AUTH_FLAGS += --set model.anthropicVertexAI.projectID=$(VERTEX_PROJECT_ID)
endif

HELM_FEATURE_FLAGS := --set clickhouse.enabled=$(CLICKHOUSE)

.PHONY: test lint clean \
	gateway-build gateway-image \
	ingest-build ingest-image \
	compose-up sync-prompts \
	cluster-up cluster-down studio-up studio-down studio-template \
	workbench-build workbench-dev \
	deploy oauth-secret

test:
	go test -v -race -cover ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

sync-prompts:
	@cp agents/platform.md charts/complytime-studio/agents/platform.md
	@for agent in threat-modeler gap-analyst policy-composer; do \
		cp agents/$$agent/prompt.md charts/complytime-studio/agents/$$agent/prompt.md; \
	done

gateway-build:
	go build -o bin/studio-gateway ./cmd/gateway/

gateway-image: workbench-build
	docker build --no-cache -f Dockerfile.gateway -t $(GATEWAY_IMAGE):$(GATEWAY_TAG) .

ingest-build:
	go build -o bin/studio-ingest ./cmd/ingest/

ingest-image:
	docker build -f Dockerfile.ingest -t studio-ingest:local .

compose-up:
	docker compose up --build

workbench-build:
	cd workbench && npm run build

workbench-dev: workbench-build gateway-build

cluster-up:
	@./deploy/kind/setup.sh

cluster-down:
	kind delete cluster --name complytime-studio

studio-up: sync-prompts
	helm upgrade --install complytime-studio ./charts/complytime-studio \
		--namespace $(NAMESPACE) \
		--set "gateway.image.repository=$(GATEWAY_IMAGE)" \
		--set "gateway.image.tag=$(GATEWAY_TAG)" \
		--set "model.provider=$${MODEL_PROVIDER:-AnthropicVertexAI}" \
		--set "model.name=$${MODEL_NAME:-claude-sonnet-4-20250514}" \
		$(HELM_AUTH_FLAGS) \
		$(HELM_FEATURE_FLAGS) \
		--timeout 5m
	@echo "Chart installed. Access: kubectl port-forward -n $(NAMESPACE) svc/studio-gateway $(PORT):8080"

studio-down:
	helm uninstall complytime-studio --namespace $(NAMESPACE)

studio-template: sync-prompts
	helm template complytime-studio ./charts/complytime-studio \
		--namespace $(NAMESPACE) \
		$(HELM_FEATURE_FLAGS)

# Create the Kubernetes secret for GitHub OAuth credentials.
# Usage: GITHUB_CLIENT_SECRET=<secret> make oauth-secret
oauth-secret:
	@if [ -z "$$GITHUB_CLIENT_SECRET" ]; then \
		echo "error: GITHUB_CLIENT_SECRET is required"; exit 1; \
	fi
	kubectl create secret generic studio-oauth-credentials \
		--namespace $(NAMESPACE) \
		--from-literal=client-secret="$$GITHUB_CLIENT_SECRET" \
		--dry-run=client -o yaml | kubectl apply -f -
	@echo "Secret studio-oauth-credentials written to namespace $(NAMESPACE)"

# Full build → load → deploy → port-forward cycle for kind clusters.
# Usage: make deploy
# With OAuth: GITHUB_CLIENT_ID=<id> GITHUB_CLIENT_SECRET=<secret> make deploy
deploy: gateway-image
	kind load docker-image $(GATEWAY_IMAGE):$(GATEWAY_TAG) --name $(KIND_CLUSTER)
	@docker exec $(KIND_CLUSTER)-control-plane \
		ctr --namespace=k8s.io images tag --force \
		localhost/$(GATEWAY_IMAGE):$(GATEWAY_TAG) \
		docker.io/library/$(GATEWAY_IMAGE):$(GATEWAY_TAG) 2>/dev/null || true
	@if [ -n "$$GITHUB_CLIENT_SECRET" ]; then \
		$(MAKE) oauth-secret; \
	fi
	$(MAKE) studio-up
	kubectl rollout restart deployment/studio-gateway -n $(NAMESPACE)
	kubectl rollout status deployment/studio-gateway -n $(NAMESPACE) --timeout=60s
	@echo "Gateway deployed. Run: kubectl port-forward -n $(NAMESPACE) svc/studio-gateway $(PORT):8080"

