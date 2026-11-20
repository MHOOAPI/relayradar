SHELL := /bin/sh
COMPOSE := docker compose

ifeq ($(OS),Windows_NT)
DEV := scripts/dev.cmd
else
DEV := ./scripts/dev.sh
endif

.PHONY: dev up build web api logs ps stop test ci

dev:
	$(DEV)

up:
	$(COMPOSE) up -d

build:
	cd web && npm ci && npm run build
	$(COMPOSE) build

web:
	cd web && npm run build

api:
	cd backend && go build ./...

logs:
	$(COMPOSE) logs -f --tail=100

ps:
	$(COMPOSE) ps

stop:
	$(COMPOSE) down

test:
	cd backend && go vet ./... && go test ./...
	cd web && npm run build

ci:
	cd backend && go vet ./... && go test ./...
	cd web && npm ci && npm run build
	docker compose config --quiet
	docker compose build
