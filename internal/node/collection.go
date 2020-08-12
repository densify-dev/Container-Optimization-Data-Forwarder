/*
Used for collecting metric data. Mostly the same as the container collection but less checks.
(pods and containers)
*/

//Package node used for collecting node metric data
package node

import (
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/common"
	"github.com/prometheus/common/model"
)

//Gets node metrics from prometheus (and checks to see if they are valid)
func getNodeMetric(result model.Value, node model.LabelName, metric string) {

	if result == nil {
		return
	}
	//Loop through the different entities in the results.
	for i := 0; i < result.(model.Matrix).Len(); i++ {
		nodeValue, ok := result.(model.Matrix)[i].Metric[node]
		if !ok {
			continue
		}
		if _, ok := nodes[string(nodeValue)]; !ok {
			continue
		}
		//validates that the value of the entity is set and if not will default to 0
		var value int
		if len(result.(model.Matrix)[i].Values) != 0 {
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
			case "cpuLimit":
				nodes[string(nodeValue)].cpuLimit = int(value)
			case "cpuRequest":
				nodes[string(nodeValue)].cpuRequest = int(value)
			case "memLimit":
				nodes[string(nodeValue)].memLimit = int(value)
			case "memRequest":
				nodes[string(nodeValue)].memRequest = int(value)
			}
		}
	}
}

//getNodeMetricString is used to parse the label based results from Prometheus related to Container Entities and store them in the systems data structure.
func getNodeMetricString(result model.Value, node model.LabelName) {
	//Validate there is data in the results.
	if result == nil {
		return
	}
	//Loop through the different entities in the results.
	for i := 0; i < result.(model.Matrix).Len(); i++ {
		nodeValue, ok := result.(model.Matrix)[i].Metric[node]
		if !ok {
			continue
		}
		if _, ok := nodes[string(nodeValue)]; !ok {
			continue
		}
		for key, value := range result.(model.Matrix)[i].Metric {
			common.AddToLabelMap(string(key), string(value), nodes[string(nodeValue)].labelMap)
		}
	}
}
