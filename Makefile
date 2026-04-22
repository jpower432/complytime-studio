# SPDX-License-Identifier: Apache-2.0

PORT ?= 8080
KIND_CLUSTER ?= complytime-studio
NAMESPACE ?= kagent
GATEWAY_IMAGE ?= studio-gateway
GATEWAY_TAG ?= local
ASSISTANT_IMAGE ?= studio-assistant
ASSISTANT_TAG ?= local

CLICKHOUSE ?= true

HELM_AUTH_FLAGS :=
ifdef GOOGLE_CLIENT_ID
HELM_AUTH_FLAGS += --set auth.google.clientId=$(GOOGLE_CLIENT_ID)
endif
ifdef VERTEX_PROJECT_ID
HELM_AUTH_FLAGS += --set model.vertexAI.projectID=$(VERTEX_PROJECT_ID)
endif

HELM_AGENT_FLAGS :=
ifdef ASSISTANT_MODEL_PROVIDER
HELM_AGENT_FLAGS += --set agents.assistant.model.provider=$(ASSISTANT_MODEL_PROVIDER)
endif
ifdef ASSISTANT_MODEL_NAME
HELM_AGENT_FLAGS += --set agents.assistant.model.name=$(ASSISTANT_MODEL_NAME)
endif

HELM_FEATURE_FLAGS := --set clickhouse.enabled=$(CLICKHOUSE)

.PHONY: test lint clean \
	gateway-build gateway-image \
	assistant-image \
	compose-up sync-prompts seed \
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
	@mkdir -p charts/complytime-studio/agents/assistant
	@cp agents/assistant/prompt.md charts/complytime-studio/agents/assistant/prompt.md

gateway-build:
	go build -o bin/studio-gateway ./cmd/gateway/

gateway-image: workbench-build
	docker build --no-cache -f Dockerfile.gateway -t $(GATEWAY_IMAGE):$(GATEWAY_TAG) .

assistant-image:
	docker build --no-cache -f agents/assistant/Dockerfile -t $(ASSISTANT_IMAGE):$(ASSISTANT_TAG) agents/assistant/

compose-up:
	docker compose up --build

workbench-build:
	cd workbench && npm run build

workbench-dev: workbench-build gateway-build

cluster-up:
	@./deploy/kind/setup.sh

cluster-down:
	kind delete cluster --name complytime-studio

HELM_MODEL_FLAGS :=
ifdef MODEL_PROVIDER
HELM_MODEL_FLAGS += --set model.provider=$(MODEL_PROVIDER)
endif
ifdef MODEL_NAME
HELM_MODEL_FLAGS += --set model.name=$(MODEL_NAME)
endif

studio-up: sync-prompts
	helm upgrade --install complytime-studio ./charts/complytime-studio \
		--namespace $(NAMESPACE) \
		--set "gateway.image.repository=$(GATEWAY_IMAGE)" \
		--set "gateway.image.tag=$(GATEWAY_TAG)" \
		--set "assistant.image.repository=$(ASSISTANT_IMAGE)" \
		--set "assistant.image.tag=$(ASSISTANT_TAG)" \
		$(HELM_MODEL_FLAGS) \
		$(HELM_AUTH_FLAGS) \
		$(HELM_AGENT_FLAGS) \
		$(HELM_FEATURE_FLAGS) \
		--timeout 5m
	@echo "Chart installed. Access: kubectl port-forward -n $(NAMESPACE) svc/studio-gateway $(PORT):8080"

studio-down:
	helm uninstall complytime-studio --namespace $(NAMESPACE)

studio-template: sync-prompts
	helm template complytime-studio ./charts/complytime-studio \
		--namespace $(NAMESPACE) \
		$(HELM_FEATURE_FLAGS)

# Create the Kubernetes secret for Google OAuth credentials.
# Usage: GOOGLE_CLIENT_SECRET=<secret> make oauth-secret
oauth-secret:
	@if [ -z "$$GOOGLE_CLIENT_SECRET" ]; then \
		echo "error: GOOGLE_CLIENT_SECRET is required"; exit 1; \
	fi
	kubectl create secret generic studio-oauth-credentials \
		--namespace $(NAMESPACE) \
		--from-literal=client-secret="$$GOOGLE_CLIENT_SECRET" \
		--dry-run=client -o yaml | kubectl apply -f -
	@echo "Secret studio-oauth-credentials written to namespace $(NAMESPACE)"

# Full build → load → deploy → port-forward cycle for kind clusters.
# Usage: make deploy
# With OAuth: GOOGLE_CLIENT_ID=<id> GOOGLE_CLIENT_SECRET=<secret> make deploy
deploy: gateway-image assistant-image
	kind load docker-image $(GATEWAY_IMAGE):$(GATEWAY_TAG) --name $(KIND_CLUSTER)
	@docker exec $(KIND_CLUSTER)-control-plane \
		ctr --namespace=k8s.io images tag --force \
		localhost/$(GATEWAY_IMAGE):$(GATEWAY_TAG) \
		docker.io/library/$(GATEWAY_IMAGE):$(GATEWAY_TAG) 2>/dev/null || true
	kind load docker-image $(ASSISTANT_IMAGE):$(ASSISTANT_TAG) --name $(KIND_CLUSTER)
	@docker exec $(KIND_CLUSTER)-control-plane \
		ctr --namespace=k8s.io images tag --force \
		localhost/$(ASSISTANT_IMAGE):$(ASSISTANT_TAG) \
		docker.io/library/$(ASSISTANT_IMAGE):$(ASSISTANT_TAG) 2>/dev/null || true
	@if [ -n "$$GOOGLE_CLIENT_SECRET" ]; then \
		$(MAKE) oauth-secret; \
	fi
	$(MAKE) studio-up
	kubectl rollout restart deployment/studio-gateway -n $(NAMESPACE)
	kubectl rollout status deployment/studio-gateway -n $(NAMESPACE) --timeout=240s
	kubectl rollout restart deployment/studio-assistant -n $(NAMESPACE)
	kubectl rollout status deployment/studio-assistant -n $(NAMESPACE) --timeout=120s
	@echo "Deployed. Run: kubectl port-forward -n $(NAMESPACE) svc/studio-gateway $(PORT):8080"

# Seed demo data into a running Studio instance.
# Requires: kubectl port-forward -n kagent svc/studio-gateway 8080:8080
seed:
	GATEWAY_URL=http://localhost:$(PORT) ./demo/seed.sh

