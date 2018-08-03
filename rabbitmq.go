package main

import (
	"fmt"
	"github.com/caarlos0/env"
	"github.com/jordanbcooper/rabbit-hole"
	sdkargs "github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/infra-integrations-sdk/metric"
	"github.com/newrelic/infra-integrations-sdk/sdk"
	"net/url"
	//	"os"
	"strconv"
	"sync"
	//"encoding/json"
)

type argumentList struct {
	sdkargs.DefaultArgumentList
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

	integration, err := sdk.NewIntegration(integrationName, integrationVersion, &args)
	fatalIfErr(err)
	log.SetupLogging(args.Verbose)

	if args.All || args.Inventory {

		populateInventory(integration.Inventory)

	}

	if args.All || args.Metrics {

		sample := integration.NewMetricSet("RabbitMQ_Sample")

		populateMetrics(sample)

	}

	fatalIfErr(integration.Publish())
}

func rmqClient() *rabbithole.Client {
	cfg := Config{}
	env.Parse(&cfg)
	rmqc, err := rabbithole.NewClient(cfg.Host, cfg.User, cfg.Password)
	fatalIfErr(err)

	return rmqc
}

func populateInventory(inventory sdk.Inventory) {
	rmqc := rmqClient()
	res, err := rmqc.Overview()
	fatalIfErr(err)

	inventory.SetItem("Software Version", "value", res.ManagementVersion)
}
func worker(mutex *sync.Mutex, rmqc *rabbithole.Client, ms *metric.MetricSet, workerId int, jobs <-chan int, results chan<- int) {
	for j := range jobs {
		fmt.Println(fmt.Sprintf("Starting page %d worker %d", j, workerId))
		values := url.Values{"page": {strconv.Itoa(j)}}
		rs, err := rmqc.PagedListQueuesWithParameters(values)
		if err != nil {
			fmt.Println(err.Error())
		}
		for _, queue := range rs.Items {
			vhostQueue := queue.Vhost + "/" + queue.Name
			// fmt.Println(vhostQueue, queue.Messages) // uncomment to log queues stats
			mutex.Lock() // We're using a mutex because the version of infra sdk uses map that is not threadsafe
			ms.SetMetric(vhostQueue, queue.Messages, metric.GAUGE)
			mutex.Unlock()
		}
		fmt.Println(fmt.Sprintf("Finishing page %d worker %d", j, workerId))
		results <- j
	}
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

	values := url.Values{"page": {"1"}}
	// values := url.Values{}
	qs, err := rmqc.PagedListQueuesWithParameters(values)
	if err != nil {
		fmt.Println(err.Error())
	}
	results := make(chan int, qs.PageCount)

	if qs.PageCount == 0 {
		fmt.Println("no queues")
		return
	}
	var mutex = &sync.Mutex{}
	// TODO allow QueueFetchWorkerCount to be configurable in boshrelease
	// Should default to 1 in the boshrelease
	cfg := Config{}
	env.Parse(&cfg)
	workerCount := cfg.Workers
	if workerCount > qs.PageCount {
		workerCount = qs.PageCount
	}
	jobs := make(chan int, workerCount)
	fmt.Println("Starting to fetch queue stats with worker count of: ", workerCount)
	for w := 1; w <= workerCount; w++ {
		go worker(mutex, rmqc, ms, w, jobs, results)

	}
	for currentPage := 1; currentPage <= qs.PageCount; currentPage++ {
		fmt.Println("Sending queue page collector job number: ", currentPage)
		jobs <- currentPage
	}
	close(jobs)

	for a := 1; a <= qs.PageCount; a++ {
		<-results
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

func fatalIfErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
