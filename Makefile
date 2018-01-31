# NewRelicIntegration RabbitMQ

# Help Helper matches comments at the start of the task block so make help gives users information about each task
.PHONY: help
help: ## Displays information about available make tasks
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: init
init: ## Installs development tooling, for active developers of this integration
	@echo "Install  GVT for dependency Management"
	go get -u github.com/FiloSottile/gvt

.PHONY: build
build: ## Compiles the integration binary and puts it in ./bin
	go build -o ./integration/bin/rabbitmq_integration

# Undocument by the help command. This helper method creates the dev environment for us
.PHONY: dev-env
dev-env:
	docker build -t newrelic-rabbitmq:dev  .

.PHONY: dev
dev: stop build dev-env ## Runs a dev container with the integration binary shared so it can be actively developed
	docker run -d \
		--hostname nrrmq-dev-$(shell hostname) \
		--name nrrmq-dev \
		-p 15672:15672 \
		-e NRIA_LICENSE_KEY=$(NRIA_LICENSE_KEY) \
		-v "$$(pwd)/integration:/var/db/newrelic-infra/custom-integrations" \
		-v "$$(pwd)/config:/etc/newrelic-infra/integrations.d" \
		newrelic-rabbitmq:dev

.PHONY: stop
stop: ## Destroys the active dev container, ignores error if container doesn't exist
	-docker rm -f nrrmq-dev 2>/dev/null

.PHONY: purge
purge: stop ## Destroys the active dev container

.PHONY: logs
logs: ## Output the dev container log
	docker logs nrrmq-dev

.PHONY: restart-newrelic
restart-newrelic: ## Restart the newrelic-infra agent on the dev container (used when changing config files)
	@echo "restarting newrelic-infra..."
	docker exec nrrmq-dev /bin/bash -c "initctl restart newrelic-infra"

.PHONY: status
status: ## Show running newrelic-integration-rabbitmq containers
	docker ps -f name=nrrmq