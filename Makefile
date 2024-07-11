COMPOSE_FILE = docker/docker-compose.yml

ifneq ("$(wildcard ./docker/docker-compose.yaml)","")
    COMPOSE_FILE = docker/docker-compose.yaml
endif

# Run docker compose, if it doesn't exist run docker-compose insted
DOCKER_COMPOSE_COMMAND := $(shell if command -v docker-compose >/dev/null 2>&1; then echo docker-compose; else echo docker compose; fi)

.PHONY: dev all

dev:
	@echo "Running docker compose in watch mode..."
	$(DOCKER_COMPOSE_COMMAND) -f $(COMPOSE_FILE) watch

all:
	@echo "Building and running docker compose in detached mode..."
	$(DOCKER_COMPOSE_COMMAND) -f $(COMPOSE_FILE) up --build -d

stop:
	@echo "Stopping docker compose..."
	$(DOCKER_COMPOSE_COMMAND) -f $(COMPOSE_FILE) down -v
	