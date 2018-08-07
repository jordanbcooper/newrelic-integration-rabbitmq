# newrelic-integration-rabbitmq
Integration for New Relic Infrastructure written in Go

This integration will be used with a Bosh Release, but I have been testing it out with a local rabbitmq in Docker running newrelic-infra agent.

[New Relic Infrastructure Agent](https://docs.newrelic.com/docs/infrastructure/new-relic-infrastructure/installation/install-infrastructure-linux)

[New Relic Infrastructure Agent SDK](https://github.com/newrelic/infra-integrations-sdk)

[Rabbit-Hole](https://github.com/michaelklishin/rabbit-hole) 

[Rabbit-Hole (Fork)](https://github.com/jordanbcooper/rabbit-hole)
< Used by this repo

## Usage

### Quick Start
1. Clone this repo into your `$GOPATH` manually, or with go get: `go get github.com/jordanbcooper/newrelic-integration-rabbitmq`
1. Ensure you've exported the `NRIA_LICENSE_KEY` environment variable and set it to your NewRelic license key. You can also pass this before a make command if you'd rather not set it permanently: `NRIA_LICENSE_KEY=somekey make dev` **THE CONTAINER WILL NOT RUN UNLESS A VALID NEWRELIC LICENSE IS PROVIDED**
1. Run `make dev` to start up a Docker container with the integration running in it
1. Run `make status` to see if your container is running. If it's not, use `make logs` to diagnose the problem, and make sure you passed in a valid NewRelic license per step two.

### Using the Makefile
This repo contains a Makefile to make active development and testing easier. Run `make help` to see what it can do:

```
help                           Displays information about available make tasks
init                           Ensures that gvt is installed for dependency management and sets up build directories
build                          Compiles the integration binary and puts it in ./bin
dev                            Runs a dev container with the integration binary shared so it can be actively developed
stop                           Destroys the active dev container, ignores error if container doesn't exist
logs                           Output the dev container log
restart-newrelic               Restart the newrelic-infra agent on the dev container (used when changing config files)
status                         Show running newrelic-integration-rabbitmq containers
```

### init
You only need to run `make init` if you're planning to actively develop this project.  It installs `gvt` for vendoring 
go dependencies.

### build
Running `make build` will compile the integration binary and put it in `./bin`

### dev
Running `make dev` will build and launch a Docker container running rabbitmq and the newrelic-infra agent. This container
is mapped to the `bin` and `config` directories of this project. Once it is running you can rebuild the binary with
`make build` and the container will automatically fetch the latest code. If the configuration files are changed you'll
need to manually restart the newrelic-infra agent which can be done with `make restart-newrelic`

### logs
Running `make logs` will show the latest logs from the dev container. It's equivalent to running `docker logs`

### stop
Stops and destroys the running dev container.

### restart-newrelic
Running `make restart-newrelic` will restart the newrelic-infra agent running on the machine using `docker exec`. This is
necessary during active development after changing any of the files in the `config` directory.

### status
Running `make status` will show you any running dev containers.

## New Relic Insights Dashboard NRQL query
Object Totals (Average):
<br>
```SELECT average(Exchanges), average(Consumers), average(Channels), average(Connections) from OrgRabbitMQ_IntegrationSample since 30 minutes ago TIMESERIES AUTO```

<br>

## Expected Output
```
{
  "name": "com.org.rabbitmq",
  "protocol_version": "2",
  "integration_version": "0.1.0",
  "data": [
    {
      "entity": {
        "name": "rabbit@localhost",
        "type": "rabbitmq_overview"
      },
      "metrics": [
        {
          "Channels": 480,
          "Connections": 117,
          "Consumers": 286,
          "Deliver": 17.4,
          "Exchanges": 44,
          "Messages": 0,
          "Messages Ready": 0,
          "Messages Unacknowledged": 0,
          "Node 0 Erlang Processes Total": 1048576,
          "Node 0 Erlang Processes Used": 441,
          "Node 0 File Descriptors Total": 300000,
          "Node 0 File Descriptors Used": 63,
          "Node 1 Erlang Processes Total": 1048576,
          "Node 1 Erlang Processes Used": 1606,
          "Node 1 File Descriptors Total": 300000,
          "Node 1 File Descriptors Used": 95,
          "Node 2 Erlang Processes Total": 1048576,
          "Node 2 Erlang Processes Used": 3987,
          "Node 2 File Descriptors Total": 300000,
          "Node 2 File Descriptors Used": 190,
          "Publish": 19,
          "Queues": 302,
          "Running": 3,
          "event_type": "RabbitMQ_Overview"
        }
      ],
      "inventory": {
        "Software Version": {
          "value": "3.6.15"
        }
      },
      "events": []
    },
    {
      "entity": {
        "name": "vhost/queue",
        "type": "queue"
      },
      "metrics": [
        {
          "consumers": 1,
          "event_type": "Rabbitmq_Queues",
          "message_rate": 0,
          "messages": 0,
          "messages_ready": 0,
          "messages_unacknowledged": 0
        }
      ],
      "inventory": {},
      "events": []
    }
   ]
 }
```
