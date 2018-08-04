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

	entity := i.LocalEntity()

	if args.All() || args.Inventory {

		populateInventory(entity.Inventory)

	}

	if args.All() || args.Metrics {

		sample := entity.NewMetricSet("RabbitMQ_Sample")
		queues := entity.NewMetricSet("RabbitMQ_Queues")
		rates := entity.NewMetricSet("RabbitMQ_QueueRates")
		populateMetrics(sample)
		populateQueues(queues)
		populateQueueMessageRates(rates)

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
func worker(rmqc *rabbithole.Client, ms *metric.Set, workerId int, jobs <-chan int, results chan<- int) {
	for j := range jobs {
		values := url.Values{"page": {strconv.Itoa(j)}}
		rs, err := rmqc.PagedListQueuesWithParameters(values)
		if err != nil {
			panicOnErr(err)
		}
		for _, queue := range rs.Items {
			vhostQueue := queue.Vhost + "/" + queue.Name
			ms.SetMetric(vhostQueue, queue.Messages, metric.GAUGE)
		}
		results <- j
	}
}

func worker2(rmqc *rabbithole.Client, ms *metric.Set, workerId int, jobs2 <-chan int, results2 chan<- int) {
	for j2 := range jobs2 {

		values := url.Values{"page": {strconv.Itoa(j2)}}
		rs, err := rmqc.PagedListQueuesWithParameters(values)
		if err != nil {
			panicOnErr(err)
		}
		for _, queue := range rs.Items {
			vhostQueue := queue.Vhost + "/" + queue.Name
			ms.SetMetric(vhostQueue, queue.MessagesDetails.Rate, metric.GAUGE)
		}
		results2 <- j2
	}
}

func populateQueues(ms *metric.Set) {
	rmqc := rmqClient()
	values := url.Values{"page": {"1"}}
	// values := url.Values{}
	qs, err := rmqc.PagedListQueuesWithParameters(values)
	if err != nil {
		panicOnErr(err)
	}
	results := make(chan int, qs.PageCount)

	if qs.PageCount == 0 {
		fmt.Println("no queues")
		return
	}
	// TODO allow QueueFetchWorkerCount to be configurable in boshrelease
	// Should default to 1 in the boshrelease
	cfg := Config{}
	env.Parse(&cfg)
	workerCount := cfg.Workers
	if workerCount > qs.PageCount {
		workerCount = qs.PageCount
	}

	jobs := make(chan int, workerCount)
	for w := 1; w <= workerCount; w++ {
		go worker(rmqc, ms, w, jobs, results)

	}
	for currentPage := 1; currentPage <= qs.PageCount; currentPage++ {
		jobs <- currentPage
	}
	close(jobs)

	for a := 1; a <= qs.PageCount; a++ {
		<-results
	}
}

func populateQueueMessageRates(ms *metric.Set) {
	rmqc := rmqClient()
	values := url.Values{"page": {"1"}}
	// values := url.Values{}
	qs, err := rmqc.PagedListQueuesWithParameters(values)
	if err != nil {
		panicOnErr(err)
	}
	results2 := make(chan int, qs.PageCount)

	if qs.PageCount == 0 {
		fmt.Println("no queues")
		return
	}
	// TODO allow QueueFetchWorkerCount to be configurable in boshrelease
	// Should default to 1 in the boshrelease
	cfg := Config{}
	env.Parse(&cfg)
	workerCount := cfg.Workers
	if workerCount > qs.PageCount {
		workerCount = qs.PageCount
	}

	jobs2 := make(chan int, workerCount)
	for w2 := 1; w2 <= workerCount; w2++ {
		go worker(rmqc, ms, w2, jobs2, results2)

	}
	for currentPage := 1; currentPage <= qs.PageCount; currentPage++ {
		jobs2 <- currentPage
	}
	close(jobs2)

	for a := 1; a <= qs.PageCount; a++ {
		<-results2
	}
}

func populateMetrics(ms *metric.Set) {
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

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
