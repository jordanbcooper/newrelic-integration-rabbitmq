package main

import (
	"fmt"
	"github.com/caarlos0/env"
	"github.com/jordanbcooper/rabbit-hole"
	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/infra-integrations-sdk/data/inventory"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"net/url"
	"strconv"
)

type argumentList struct {
	sdkArgs.DefaultArgumentList
}

type Config struct {
	Workers  int    `env:"QUEUE_FETCH_WORKER_COUNT"`
	User     string `env:"RMQ_USERNAME"`
	Password string `env:"RMQ_PASSWORD"`
	Host     string `env:"RMQ_HOSTNAME"`
}

const (
	integrationName    = "com.org.rabbitmq"
	integrationVersion = "0.1.0"
)

var args argumentList

func main() {

	i, err := integration.New(integrationName, integrationVersion, integration.Args(&args))
	panicOnErr(err)
	rmqc := rmqClient()
	cn, err := rmqc.GetClusterName()
	panicOnErr(err)
	entityOverview, err := i.Entity(cn.Name, "rabbitmq_overview")
	panicOnErr(err)

	if args.All() || args.Inventory {

		populateInventory(entityOverview.Inventory)

	}

	if args.All() || args.Metrics {
		// api/overview
		overview := entityOverview.NewMetricSet("RabbitMQ_Overview")
		populateOverview(overview)
		// Queue Messages
		// Generate Entities
		values := url.Values{"page": {"1"}}
		qs, err := rmqc.PagedListQueuesWithParameters(values)
		panicOnErr(err)

		for currentPage := 1; currentPage <= qs.PageCount; currentPage++ {
			values := url.Values{"page": {strconv.Itoa(currentPage)}}
			rs, err := rmqc.PagedListQueuesWithParameters(values)
			panicOnErr(err)
			for _, queue := range rs.Items {
				vhostQueue := queue.Vhost + "/" + queue.Name
				entityQueues, err := i.Entity(vhostQueue, "rabbitmq_queue")
				panicOnErr(err)
				queues := entityQueues.NewMetricSet("Rabbitmq_Queues")
				populateQueues(queues)
			}
		}

	}
	panicOnErr(i.Publish())
}

func rmqClient() *rabbithole.Client {
	cfg := Config{}
	env.Parse(&cfg)
	rmqc, err := rabbithole.NewClient(cfg.Host, cfg.User, cfg.Password)
	panicOnErr(err)

	return rmqc
}

func populateInventory(i *inventory.Inventory) {
	rmqc := rmqClient()
	res, err := rmqc.Overview()
	panicOnErr(err)

	i.SetItem("Software Version", "value", res.ManagementVersion)
}

func populateOverview(ms *metric.Set) {
	rmqc := rmqClient()
	res, err := rmqc.Overview()
	panicOnErr(err)
	xs, err := rmqc.ListNodes()
	//Cluster Running Count (GET ME INTO A FUNCTION!)
	var runCount = 0
	var nodeCount = len(xs)
	var i = 0
	for i <= nodeCount-1 {
		var nodeIsRunning = xs[i].IsRunning

		if nodeIsRunning {
			runCount = runCount + 1
			fduVar := fmt.Sprintf("Node %v File Descriptors Used", i)
			fdtVar := fmt.Sprintf("Node %v File Descriptors Total", i)
			procuVar := fmt.Sprintf("Node %v Erlang Processes Used", i)
			proctVar := fmt.Sprintf("Node %v Erlang Processes Total", i)
			ms.SetMetric(fduVar, xs[i].FdUsed, metric.GAUGE)
			ms.SetMetric(fdtVar, xs[i].FdTotal, metric.GAUGE)
			ms.SetMetric(procuVar, xs[i].ProcUsed, metric.GAUGE)
			ms.SetMetric(proctVar, xs[i].ProcTotal, metric.GAUGE)
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
	ms.SetMetric("Publish", res.MessageStats.PublishDetails.Rate, metric.GAUGE)
	ms.SetMetric("Deliver", res.MessageStats.DeliverDetails.Rate, metric.GAUGE)
	//Cluster Status
	ms.SetMetric("Running", runCount, metric.GAUGE)

}

func populateQueues(queues *metric.Set) {
	rmqc := rmqClient()
	values := url.Values{"page": {"1"}}
	// values := url.Values{}
	qs, err := rmqc.PagedListQueuesWithParameters(values)
	panicOnErr(err)
	for currentPage := 1; currentPage <= qs.PageCount; currentPage++ {
		values := url.Values{"page": {strconv.Itoa(currentPage)}}
		rs, err := rmqc.PagedListQueuesWithParameters(values)
		panicOnErr(err)
		for _, queue := range rs.Items {
			queues.SetMetric("messages", queue.Messages, metric.GAUGE)
			queues.SetMetric("consumers", queue.Consumers, metric.GAUGE)
			queues.SetMetric("message_rate", queue.MessagesDetails.Rate, metric.GAUGE)
			queues.SetMetric("messages_ready", queue.MessagesReady, metric.GAUGE)
			queues.SetMetric("messages_unacknowledged", queue.MessagesUnacknowledged, metric.GAUGE)
		}

	}
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
