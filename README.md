# newrelic-integration-rabbitmq
Integration for New Relic Infrastructure written in Go

This integration will be used with a Bosh Release, but I have been testing it out with a local rabbitmq in Docker running newrelic-infra agent.


[New Relic Infrastructure Agent](https://docs.newrelic.com/docs/infrastructure/new-relic-infrastructure/installation/install-infrastructure-linux)

[Rabbit-Hole](https://github.com/michaelklishin/rabbit-hole)


# Usage

Docker
`docker run -d --hostname my-rabbit --name some-rabbit -p 15672:15672 rabbitmq:3-management`

Install New Relic Infrastructure Agent

Build .go file

Copy rabbitmq_integration binary to /var/db/newrelic-infra/custom-integrations/bin/
Copy rabbitmq_integration-definition.yml to /var/db/newrelic-infra/custom-integrations/
Copy rabbitmq_integration-config.yml to /etc/newrelic-infra/integrations.d

`newrelic-infra start`

# Testing

`./rabbitmq_integration -pretty`


`
Output:
{
        "name": "com.myorg.rabbitmq",
        "protocol_version": "1",
        "integration_version": "0.1.0",
        "metrics": [
                {
                        "Consumers": 0,
                        "Exchanges": 8,
                        "event_type": "MyOrgRabbitMQ_IntegrationSample"
                }
        ],
        "inventory": {
                "Node": {
                        "value": "rabbit@my-rabbit"
                }
        },
        "events": []
}
`
