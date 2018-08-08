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
	Cluster  string `env:"RMQ_CLUSTER"`
}

const (
	integrationName    = "com.org.rabbitmq"
	integrationVersion = "1.0.0"
)

var args argumentList

var cfg = Config{}

func main() {

	i, err := integration.New(integrationName, integrationVersion, integration.Args(&args))
	panicOnErr(err)
	env.Parse(&cfg)
	entityOverview, err := i.Entity(cfg.Cluster, "cluster_overview")
	panicOnErr(err)

	if args.All() || args.Inventory {

		populateInventory(entityOverview.Inventory)

	}

	if args.All() || args.Metrics {
		// overview
		overview := entityOverview.NewMetricSet("RabbitMQ_Overview")
		populateOverview(overview)
		// queues
		populateQueues(i)
	}
	panicOnErr(i.Publish())
}

func rmqClient() *rabbithole.Client {
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
	var n = 0
	for n <= nodeCount-1 {
		var nodeIsRunning = xs[n].IsRunning

		if nodeIsRunning {
			runCount = runCount + 1
			fduVar := fmt.Sprintf("Node %v File Descriptors Used", n)
			fdtVar := fmt.Sprintf("Node %v File Descriptors Total", n)
			procuVar := fmt.Sprintf("Node %v Erlang Processes Used", n)
			proctVar := fmt.Sprintf("Node %v Erlang Processes Total", n)
			ms.SetMetric(fduVar, xs[n].FdUsed, metric.GAUGE)
			ms.SetMetric(fdtVar, xs[n].FdTotal, metric.GAUGE)
			ms.SetMetric(procuVar, xs[n].ProcUsed, metric.GAUGE)
			ms.SetMetric(proctVar, xs[n].ProcTotal, metric.GAUGE)
		}
		n = n + 1
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

func worker(rmqc *rabbithole.Client, i *integration.Integration, workerId int, jobs <-chan int, results chan<- int) {
	for j := range jobs {
		values := url.Values{"page": {strconv.Itoa(j)}}
		rs, err := rmqc.PagedListQueuesWithParameters(values)
		panicOnErr(err)
		for _, queue := range rs.Items {
			vhostQueue := queue.Vhost + "/" + queue.Name
			entityQueues, err := i.Entity(vhostQueue, "queue")
			panicOnErr(err)
			queues := entityQueues.NewMetricSet("Rabbitmq_Queues")
			queues.SetMetric("messages", queue.Messages, metric.GAUGE)
			queues.SetMetric("consumers", queue.Consumers, metric.GAUGE)
			queues.SetMetric("message_rate", queue.MessagesDetails.Rate, metric.GAUGE)
			queues.SetMetric("messages_ready", queue.MessagesReady, metric.GAUGE)
			queues.SetMetric("messages_unacknowledged", queue.MessagesUnacknowledged, metric.GAUGE)
		}
		results <- j
	}
}

func populateQueues(i *integration.Integration) {
	rmqc := rmqClient()
	values := url.Values{"page": {"1"}}
	// values := url.Values{}
	qs, err := rmqc.PagedListQueuesWithParameters(values)
	panicOnErr(err)
	results := make(chan int, qs.PageCount)
	if qs.PageCount == 0 {
		env.Parse(&cfg)
		noQueues := cfg.Cluster + "/no_queues"
		entityQueues, err := i.Entity(noQueues, "queue")
		panicOnErr(err)
		queues := entityQueues.NewMetricSet("Rabbitmq_Queues")
		queues.SetMetric("queues", 0, metric.GAUGE)
		return
	}
	// TODO allow QueueFetchWorkerCount to be configurable in boshrelease
	// Should default to 1 in the boshrelease
	env.Parse(&cfg)
	workerCount := cfg.Workers
	if workerCount > qs.PageCount {
		workerCount = qs.PageCount
	}

	jobs := make(chan int, workerCount)
	for w := 1; w <= workerCount; w++ {
		go worker(rmqc, i, w, jobs, results)

	}
	for currentPage := 1; currentPage <= qs.PageCount; currentPage++ {
		jobs <- currentPage
	}
	close(jobs)

	for a := 1; a <= qs.PageCount; a++ {
		<-results
	}

}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
