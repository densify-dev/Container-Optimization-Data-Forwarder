/*
Used for collecting metric data. Mostly the same as the container collection but less checks.
(namespace, pods and containers)
*/

//Package node used for collecting node metric data
package node

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
	"github.com/prometheus/common/model"
)

//Gets node metrics from prometheus (and checks to see if they are valid)
func getNodeMetric(result model.Value, namespace, node model.LabelName, metric string) {

	if result != nil {
		//Loop through the different entities in the results.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our systems structure.
			if nodeValue, ok := result.(model.Matrix)[i].Metric[node]; ok {
				if _, ok := nodes[string(nodeValue)]; ok {
					//validates that the value of the entity is set and if not will default to 0
					var value int
					if len(result.(model.Matrix)[i].Values) == 0 {
						value = 0
					} else {
						value = int(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
					}

					//Check which metric this is for and update the corresponding variable for this container in the system data structure
					if metric == "capacity" {
						capacityType := result.(model.Matrix)[i].Metric["resource"]
						switch capacityType {
						case "cpu":
							nodes[string(nodeValue)].cpuCapacity = int(value)
						case "memory":
							nodes[string(nodeValue)].memCapacity = int(value)
						case "pods":
							nodes[string(nodeValue)].podsCapacity = int(value)
						case "ephemeral_storage":
							nodes[string(nodeValue)].ephemeralStorageCapacity = int(value)
						case "hugepages_2Mi":
							nodes[string(nodeValue)].hugepages2MiCapacity = int(value)
						}
					} else if metric == "allocatable" {
						capacityType := result.(model.Matrix)[i].Metric["resource"]
						switch capacityType {
						case "cpu":
							nodes[string(nodeValue)].cpuAllocatable = int(value)
						case "memory":
							nodes[string(nodeValue)].memAllocatable = int(value)
						case "pods":
							nodes[string(nodeValue)].podsAllocatable = int(value)
						case "ephemeral_storage":
							nodes[string(nodeValue)].ephemeralStorageAllocatable = int(value)
						case "hugepages_2Mi":
							nodes[string(nodeValue)].hugepages2MiAllocatable = int(value)
						}
					} else {

						switch metric {
						case "capacity_cpu":
							nodes[string(nodeValue)].cpuCapacity = int(value)
						case "capacity_mem":
							nodes[string(nodeValue)].memCapacity = int(value)
						case "capacity_pod":
							nodes[string(nodeValue)].podsCapacity = int(value)
						case "allocatable_cpu":
							nodes[string(nodeValue)].cpuAllocatable = int(value)
						case "allocatable_mem":
							nodes[string(nodeValue)].memAllocatable = int(value)
						case "allocatable_pod":
							nodes[string(nodeValue)].podsAllocatable = int(value)
						case "netSpeedBytes":
							nodes[string(nodeValue)].netSpeedBytes = int(value)
						}
					}
				}
			}
		}
	}
}

func getWorkload(promaddress, fileName, metricName, query2, aggregrator, clusterName, promAddr, interval string, intervalSize, history int, currentTime time.Time) {
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
	workloadWrite, err := os.Create("./data/node/" + aggregrator + `_` + fileName + ".csv")
	if err != nil {
		log.Println(err)
	}
	fmt.Fprintf(workloadWrite, "cluster,node,Datetime,%s\n", metricName)

	//If the History parameter is set to anything but default 1 then will loop through the calls starting with the current day\hour\minute interval and work backwards.
	//This is done as the farther you go back in time the slpwer prometheus querying becomes and we have seen cases where will not run from timeouts on Prometheus.
	//As a result if we do hit an issue with timing out on Prometheus side we still can send the current data and data going back to that point vs losing it all.
	for historyInterval = 0; int(historyInterval) < history; historyInterval++ {
		start, end = prometheus.TimeRange(interval, intervalSize, currentTime, historyInterval)
		result = prometheus.MetricCollect(promaddress, query2, start, end, "Node", metricName)
		writeWorkload(workloadWrite, result, "node", promAddr, cluster)
	}
	//Close the workload files.
	workloadWrite.Close()
}

//getNodeMetricString is used to parse the label based results from Prometheus related to Container Entities and store them in the systems data structure.
func getNodeMetricString(result model.Value, node model.LabelName, metric string) {
	var tempSystems = map[string]map[string]string{}
	//Validate there is data in the results.
	if result != nil {
		//Loop through the different entities in the results.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our temp structure if not it will be added.
			if nodeValue, ok := result.(model.Matrix)[i].Metric[node]; ok {
				if _, ok := nodes[string(nodeValue)]; ok {
					if _, ok := tempSystems[string(nodeValue)]; ok == false {
						tempSystems[string(nodeValue)] = map[string]string{}
					}
					//loop through all the labels for an entity and store them in a map. For controller based entities where there will be multiple copies of containers they will have there values concatinated together.
					for key, value := range result.(model.Matrix)[i].Metric {
						if _, ok := tempSystems[string(nodeValue)][string(key)]; ok == false {
							tempSystems[string(nodeValue)][string(key)] = strings.Replace(string(value), ",", ";", -1)
						} else {
							if strings.Contains(tempSystems[string(nodeValue)][string(key)], strings.Replace(string(value), ",", ";", -1)) {
								tempSystems[string(nodeValue)][string(key)] += ";" + strings.Replace(string(value), ",", ";", -1)
							}
						}
					}
				}
			}
		}
		//Process the temp data structure to produce 1 string that will written into specific variable in the system data structure.
		for kn := range tempSystems {
			tempAttr := ""
			for key, value := range tempSystems[kn] {
				//Validate the length of the key and value to be less then 256 characters when combined together per value in the attribute to be loaded.
				if len(key) < 250 {
					if len(value)+3+len(key) < 256 {
						tempAttr += key + " : " + value + "|"
					} else {
						templength := 256 - 3 - len(key)
						tempAttr += key + " : " + value[:templength] + "|"
					}
				}
			}
			//Write out the combined string into the variable in the systems data structure based on which metric you provided.
			tempAttr = tempAttr[:len(tempAttr)-1]
			if metric == "nodeLabel" {
				nodes[kn].nodeLabel = tempAttr
			}
		}
	}
}
