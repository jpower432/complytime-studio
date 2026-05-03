# SPDX-License-Identifier: Apache-2.0

PORT ?= 8080
KIND_CLUSTER ?= complytime-studio
NAMESPACE ?= kagent
GATEWAY_IMAGE ?= studio-gateway
GATEWAY_TAG ?= local
ASSISTANT_IMAGE ?= studio-assistant
ASSISTANT_TAG ?= local

CLICKHOUSE ?= false
NATS ?= true

HELM_AUTH_FLAGS :=
ifdef OIDC_CLIENT_ID
HELM_AUTH_FLAGS += --set auth.oauth2Proxy.clientId=$(OIDC_CLIENT_ID)
ifdef OIDC_ISSUER_URL
HELM_AUTH_FLAGS += --set auth.oauth2Proxy.issuerUrl=$(OIDC_ISSUER_URL)
endif
ifdef OIDC_CALLBACK_URL
HELM_AUTH_FLAGS += --set auth.oauth2Proxy.callbackUrl=$(OIDC_CALLBACK_URL)
endif
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

HELM_FEATURE_FLAGS := --set clickhouse.enabled=$(CLICKHOUSE) --set nats.enabled=$(NATS)

.PHONY: test lint clean \
	gateway-build gateway-image \
	assistant-image \
	compose-up sync-prompts seed \
	cluster-up cluster-down studio-up studio-down studio-template \
	workbench-build workbench-dev \
	deploy oauth-secret \
	demo demo-change demo-all

test:
	go test -v -race -cover ./...

test-integration:
	@test -n "$(POSTGRES_TEST_URL)" || (echo "POSTGRES_TEST_URL required — e.g. postgres://user:pass@localhost:5432/test?sslmode=disable" && exit 1)
	POSTGRES_TEST_URL=$(POSTGRES_TEST_URL) go test -v -race -cover ./internal/store/ ./internal/postgres/

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

sync-prompts:
	@mkdir -p charts/complytime-studio/agents/assistant
	@cp agents/assistant/prompt.md charts/complytime-studio/agents/assistant/prompt.md

sync-skills:
	@rsync -a --delete --exclude='.gitkeep' skills/ agents/assistant/skills/

gateway-build:
	go build -o bin/studio-gateway ./cmd/gateway/

gateway-image: workbench-build
	docker build --no-cache -f Dockerfile.gateway -t $(GATEWAY_IMAGE):$(GATEWAY_TAG) .

assistant-image: sync-skills
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
		--reset-values \
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

# Create the Kubernetes secret for OIDC credentials.
# Usage: OIDC_CLIENT_SECRET=<secret> make oidc-secret
oidc-secret:
	@if [ -z "$$OIDC_CLIENT_SECRET" ]; then \
		echo "error: OIDC_CLIENT_SECRET is required"; exit 1; \
	fi
	kubectl create secret generic studio-oauth-credentials \
		--namespace $(NAMESPACE) \
		--from-literal=client-secret="$$OIDC_CLIENT_SECRET" \
		--dry-run=client -o yaml | kubectl apply -f -
	@echo "Secret studio-oauth-credentials written to namespace $(NAMESPACE)"

# Full build → load → deploy → port-forward cycle for kind clusters.
# Usage: make deploy
# With OIDC: OIDC_CLIENT_ID=<id> OIDC_CLIENT_SECRET=<secret> OIDC_ISSUER_URL=<url> make deploy
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
	@if [ -n "$$OIDC_CLIENT_SECRET" ]; then \
		$(MAKE) oidc-secret; \
	fi
	$(MAKE) studio-up
	kubectl rollout restart deployment/studio-gateway -n $(NAMESPACE)
	kubectl rollout status deployment/studio-gateway -n $(NAMESPACE) --timeout=240s
	kubectl delete pods -n $(NAMESPACE) -l app.kubernetes.io/name=studio-assistant --ignore-not-found
	kubectl wait --for=condition=ready pod -n $(NAMESPACE) -l app.kubernetes.io/name=studio-assistant --timeout=120s 2>/dev/null || true
	@echo "Deployed. Run: kubectl port-forward -n $(NAMESPACE) svc/studio-gateway $(PORT):8080"

# Seed demo data into a running Studio instance.
# Port-forwards directly to the gateway container (bypassing OAuth2 Proxy).
SEED_PORT ?= 9090
STUDIO_API_TOKEN ?= studio-dev-token
seed:
	@echo "Port-forwarding to gateway pod (bypassing OAuth2 Proxy)..."
	@kubectl port-forward -n $(NAMESPACE) deployment/studio-gateway $(SEED_PORT):8080 &
	@sleep 2
	@GATEWAY_URL=http://localhost:$(SEED_PORT) STUDIO_API_TOKEN=$(STUDIO_API_TOKEN) ./demo/seed.sh; \
		EXIT_CODE=$$?; \
		kill %1 2>/dev/null; \
		exit $$EXIT_CODE

# Record the baseline SOC 2 gap analysis demo video.
# Output: demo/cypress/videos/soc2-gap-analysis.cy.js.mp4
demo:
	cd demo && npx cypress run --no-runner-ui --spec 'cypress/e2e/soc2-gap-analysis.cy.js'

# Record demo video for a specific change.
# Usage: CHANGE=generic-oidc-auth make demo-change
demo-change:
	@if [ -z "$$CHANGE" ]; then echo "error: CHANGE is required (e.g. CHANGE=generic-oidc-auth make demo-change)"; exit 1; fi
	cd demo && npx cypress run --no-runner-ui --spec "cypress/e2e/$$CHANGE-demo.cy.js"

# Record all demo videos (baseline + all change demos).
# Output: demo/cypress/videos/*.mp4
demo-all:
	cd demo && npx cypress run --no-runner-ui --spec 'cypress/e2e/*.cy.js'

