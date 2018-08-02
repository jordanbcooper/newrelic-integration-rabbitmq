package main

import (
	"fmt"
	"github.com/jordanbcooper/rabbit-hole"
	sdkargs "github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/infra-integrations-sdk/metric"
	"github.com/newrelic/infra-integrations-sdk/sdk"
	"net/url"
	"os"
	"strconv"
	//"encoding/json"
)

type argumentList struct {
	sdkargs.DefaultArgumentList
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

	inventory.SetItem("Software Version", "value", res.ManagementVersion)
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
	jobs := make(chan int, qs.PageCount)

	workerCount := 5
	if workerCount > qs.PageCount {
		workerCount = qs.PageCount
	}

	for w := 1; w <= workerCount; w++ {
		go func(jobs <-chan int) {
			fmt.Println("Starting jobs")
			for j := range jobs {
				fmt.Println("String", j)
				values := url.Values{"page": {strconv.Itoa(j)}}
				rs, err := rmqc.PagedListQueuesWithParameters(values)
				if err != nil {
					fmt.Println(err.Error())
				}
				fmt.Println(rs.Items, "Test")
				for _, queue := range rs.Items {
					fmt.Println(queue.Vhost, queue.Name)
					vhostQueue := queue.Vhost + "/" + queue.Name
					fmt.Println(vhostQueue)
					ms.SetMetric(vhostQueue, queue.Messages, metric.GAUGE)
				}
			}
			fmt.Println("Job done", w)
		}(jobs)

	}
	for currentPage := 1; currentPage <= qs.PageCount; currentPage++ {
		fmt.Println(currentPage)
		jobs <- currentPage
	}
	close(jobs)
	//	for currentPage := 1; currentPage <= qs.PageCount; currentPage++ {
	//		values := url.Values{"page": {strconv.Itoa(currentPage)}}
	//		rs, err := rmqc.PagedListQueuesWithParameters(values)
	//		if err != nil {
	//			fmt.Println(err.Error())
	//		}
	//		for _, queue := range rs.Items {
	//			vhostQueue := queue.Vhost + "/" + queue.Name
	//			ms.SetMetric(vhostQueue, queue.Messages, metric.GAUGE)
	//		}
	//
	//	}

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
