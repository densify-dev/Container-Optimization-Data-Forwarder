package cluster

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
	"github.com/prometheus/common/model"
)

//Gets cluster metrics from prometheus (and checks to see if they are valid)
func getClusterMetric(result model.Value, metric string) {

	if result != nil {

		//validates that the value of the entity is set and if not will default to 0
		var value int
		if len(result.(model.Matrix)[0].Values) == 0 {
			value = 0
		} else {
			value = int(result.(model.Matrix)[0].Values[len(result.(model.Matrix)[0].Values)-1].Value)
		}

		//Check which metric this is for and update the corresponding variable for this container in the system data structure

		switch metric {
		case "cpuLimit":
			clusterEntity.cpuLimit = int(value)
		case "cpuRequest":
			clusterEntity.cpuRequest = int(value)
		case "memLimit":
			clusterEntity.memLimit = int(value)
		case "memRequest":
			clusterEntity.memRequest = int(value)
		}

	}
}

func getWorkload(promaddress, fileName, metricName, query, aggregator, clusterName, promAddr, interval string, intervalSize, history int, currentTime time.Time) {
	var query2 string
	var cluster string
	if clusterName == "" {
		cluster = promAddr
	} else {
		cluster = clusterName
	}
	var historyInterval time.Duration
	historyInterval = 0
	var result model.Value
	//var query string
	var start, end time.Time
	//Open the files that will be used for the workload data types and write out there headers.
	workloadWrite, err := os.Create("./data/cluster/" + aggregator + `_` + fileName + ".csv")
	if err != nil {
		log.Println(prometheus.LogMessage("[ERROR]", promAddr, entityKind, metricName, err.Error(), query2))
	}
	fmt.Fprintf(workloadWrite, "cluster,Datetime,%s\n", metricName)

	//If the History parameter is set to anything but default 1 then will loop through the calls starting with the current day\hour\minute interval and work backwards.
	//This is done as the farther you go back in time the slpwer prometheus querying becomes and we have seen cases where will not run from timeouts on Prometheus.
	//As a result if we do hit an issue with timing out on Prometheus side we still can send the current data and data going back to that point vs losing it all.
	for historyInterval = 0; int(historyInterval) < history; historyInterval++ {
		start, end = prometheus.TimeRange(interval, intervalSize, currentTime, historyInterval)

		query2 = aggregator + "(" + query + `)`
		result = prometheus.MetricCollect(promaddress, query2, start, end, "Cluster", metricName)
		writeWorkload(workloadWrite, result, promAddr, cluster)
	}
	//Close the workload files.
	workloadWrite.Close()
}
