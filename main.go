package main

import (
	"fmt"
	"github.com/michaelklishin/rabbit-hole"
	sdkargs "github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/infra-integrations-sdk/metric"
	"github.com/newrelic/infra-integrations-sdk/sdk"
	"os"
	//"encoding/json"
)

type argumentList struct {
	sdkargs.DefaultArgumentList
}

const (
	integrationName    = "com.myorg.rabbitmq"
	integrationVersion = "0.1.0"
)

var args argumentList

func main() {

	integration, err := sdk.NewIntegration(integrationName, integrationVersion, &args)
	fatalIfErr(err)
	log.SetupLogging(args.Verbose)

	if args.All || args.Inventory {

		populateInventory(integration.Inventory)

	}

	if args.All || args.Metrics {

		sample := integration.NewMetricSet("RabbitMQ_IntegrationSample")

		populateMetrics(sample)

	}

	fatalIfErr(integration.Publish())
}

func rmqClient() *rabbithole.Client {

	rmqUser := os.Getenv("RMQ_USERNAME")
	if rmqUser == "" {
		rmqUser = "guest"
	}

	rmqHostname := os.Getenv("RMQ_HOSTNAME")
	if rmqHostname == "" {
		rmqHostname = "http://localhost"
	}

	rmqPort := os.Getenv("RMQ_PORT")
	if rmqPort == "" {
		rmqPort = "15672"
	}

	rmqPassword := os.Getenv("RMQ_PASSWORD")
	if rmqPassword == "" {
		rmqPassword = "guest"
	}

	rmqc, err := rabbithole.NewClient(fmt.Sprintf("%s:%s", rmqHostname, rmqPort), rmqUser, rmqPassword)
	fatalIfErr(err)

	return rmqc
}

func populateInventory(inventory sdk.Inventory) {
	rmqc := rmqClient()
	res, err := rmqc.Overview()
	fatalIfErr(err)

	inventory.SetItem("Node", "value", res.Node)

}

func populateMetrics(ms *metric.MetricSet) {
	rmqc := rmqClient()
	res, err := rmqc.Overview()
	fatalIfErr(err)
	xs, err := rmqc.ListNodes()
	//Cluster Running Count (GET ME INTO A FUNCTION!)
	var runCount = 0
	var nodeCount = len(xs) 
	var i = 0
	for i <= nodeCount - 1 {
		var nodeIsRunning = xs[i].IsRunning

		if nodeIsRunning {
			runCount = runCount + 1
		}
		i = i + 1
	}

	// Object Totals
	ms.SetMetric("Exchanges", res.ObjectTotals.Exchanges, metric.GAUGE)
	ms.SetMetric("Queues", res.ObjectTotals.Queues, metric.GAUGE)
	ms.SetMetric("Connections", res.ObjectTotals.Connections, metric.GAUGE)
	ms.SetMetric("Channels", res.ObjectTotals.Channels, metric.GAUGE)
	ms.SetMetric("Consumers", res.ObjectTotals.Consumers, metric.GAUGE)
	//Queue Totals
	ms.SetMetric("Messages", res.QueueTotals.Messages, metric.GAUGE)
	ms.SetMetric("Messages Unacknowledged", res.QueueTotals.MessagesUnacknowledged, metric.GAUGE)
	ms.SetMetric("Messages Ready", res.QueueTotals.MessagesReady, metric.GAUGE)
	//Message Stats
	ms.SetMetric("Publish", res.MessageStats.Publish, metric.GAUGE)
	ms.SetMetric("Deliver", res.MessageStats.Deliver, metric.GAUGE)
	//Cluster Status
	ms.SetMetric("Running", runCount, metric.GAUGE)
}

func fatalIfErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
