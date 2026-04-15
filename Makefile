# SPDX-License-Identifier: Apache-2.0

PORT ?= 8080

.PHONY: test lint clean \
	gateway-build gateway-image \
	ingest-build ingest-image \
	compose-up sync-prompts \
	cluster-up cluster-down studio-up studio-down studio-template \
	workbench-build workbench-dev

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

gateway-image:
	docker build -f Dockerfile.gateway -t studio-gateway:local .

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
		--namespace kagent \
		--set "gateway.image.repository=studio-gateway" \
		--set "gateway.image.tag=local" \
		--set "model.provider=$${MODEL_PROVIDER:-AnthropicVertexAI}" \
		--set "model.name=$${MODEL_NAME:-claude-sonnet-4-20250514}" \
		--timeout 5m
	@echo "Chart installed. Access: kubectl port-forward -n kagent svc/studio-gateway 8080:8080"

studio-down:
	helm uninstall complytime-studio --namespace kagent

studio-template: sync-prompts
	helm template complytime-studio ./charts/complytime-studio \
		--namespace kagent
