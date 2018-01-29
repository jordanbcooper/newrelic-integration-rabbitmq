# newrelic-integration-rabbitmq
Integration for New Relic Infrastructure written in Go


[New Relic Infrastructure Agent](https://docs.newrelic.com/docs/infrastructure/new-relic-infrastructure/installation/install-infrastructure-linux)

[Rabbit-Hole](https://github.com/michaelklishin/rabbit-hole)

# Usage

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
