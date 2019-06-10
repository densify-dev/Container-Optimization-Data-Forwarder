//Package container2 collects data related to containers and formats into csv files to send to Densify.
package container2

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
	"github.com/prometheus/common/model"
)

//getContainerMetric is used to parse the results from Prometheus related to Container Entities and store them in the systems data structure.
func getContainerMetric(result model.Value, namespace model.LabelName, pod, container model.LabelName, metric string) {
	//Validate there is data in the results.
	if result != nil {
		//Loop through the different entities in the results.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our systems structure.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := systems[string(namespaceValue)]; ok {
					//Validate that the data contains the pod label with value and check it exists in our systems structure
					if podValue, ok := result.(model.Matrix)[i].Metric[pod]; ok {
						if _, ok := systems[string(namespaceValue)].pointers["Pod__"+string(podValue)]; ok {
							//Validate that the data contains the container label with value and check it exists in our systems structure
							if containerValue, ok := result.(model.Matrix)[i].Metric[container]; ok {
								if _, ok := systems[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)]; ok {
									//validates that the value of the entity is set and if not will default to 0
									var value int
									if len(result.(model.Matrix)[i].Values) == 0 {
										value = 0
									} else {
										value = int(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
									}
									//Check which metric this is for and update the corresponding variable for this container in the system data structure
									if metric == "memory" {
										systems[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].memory = value
									} else if metric == "cpuLimit" {
										systems[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].cpuLimit = value
									} else if metric == "cpuRequest" {
										systems[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].cpuRequest = value
									} else if metric == "memLimit" {
										systems[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].memLimit = value
									} else if metric == "memRequest" {
										systems[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].memRequest = value
									} else if metric == "restarts" {
										systems[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].restarts = value
									} else if metric == "powerState" {
										systems[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].powerState = value
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

//getContainerMetricString is used to parse the label based results from Prometheus related to Container Entities and store them in the systems data structure.
func getContainerMetricString(result model.Value, namespace model.LabelName, pod, container model.LabelName, metric string) {
	//Validate there is data in the results.
	if result != nil {
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our temp structure if not it will be added.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := systems[string(namespaceValue)]; ok {
					//Validate that the data contains the pod label with value and check it exists in our temp structure if not it will be added
					if podValue, ok := result.(model.Matrix)[i].Metric[pod]; ok {
						if _, ok := systems[string(namespaceValue)].pointers["Pod__"+string(podValue)]; ok {
							//Validate that the data contains the container label with value and check it exists in our temp structure if not it will be added
							if containerValue, ok := result.(model.Matrix)[i].Metric[container]; ok {
								if _, ok := systems[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)]; ok {
									//loop through all the labels for an entity and store them in a map. For controller based entities where there will be multiple copies of containers they will have there values concatinated together.
									for key, value := range result.(model.Matrix)[i].Metric {
										addToLabelMap(string(key), string(value), systems[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].labelMap)
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

//getmidMetric is used to parse the results from Prometheus related to mid Entities and store them in the systems data structure.
func getMidMetric(result model.Value, namespace model.LabelName, mid model.LabelName, metric string, prefix string) {
	//Validate there is data in the results.
	if result != nil {
		//Loop through the different entities in the results.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our systems structure.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := systems[string(namespaceValue)]; ok {
					//Validate that the data contains the mid label with value and check it exists in our systems structure
					if midValue, ok := result.(model.Matrix)[i].Metric[mid]; ok {
						if _, ok := systems[string(namespaceValue)].pointers[prefix+"__"+string(midValue)]; ok {
							if _, ok := systems[string(namespaceValue)].midLevels[prefix+"__"+string(midValue)]; ok {
								//validates that the value of the entity is set and if not will default to 0
								var value int64
								if len(result.(model.Matrix)[i].Values) == 0 {
									value = 0
								} else {
									value = int64(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
								}
								_ = value
								//Check which metric this is for and update the corresponding variable for this mid in the system data structure

								if metric == "currentSize" {
									systems[string(namespaceValue)].pointers[prefix+"__"+string(midValue)].currentSize = int(value)
								} else if metric == "creationTime" {
									systems[string(namespaceValue)].pointers[prefix+"__"+string(midValue)].creationTime = value
								} else {
									addToLabelMap(metric, strconv.FormatInt(value, 10), systems[string(namespaceValue)].pointers[prefix+"__"+string(midValue)].labelMap)
								}
							}
						}
					}
				}
			}
		}
	}
}

//getmidMetricString is used to parse the label based results from Prometheus related to mid Entities and store them in the systems data structure.
func getMidMetricString(result model.Value, namespace model.LabelName, mid model.LabelName, metric string, prefix string) {
	//temp structure used to store data while working with it. As we are combining the labels into a formatted string for loading.
	//Validate there is data in the results.
	if result != nil {
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our temp structure if not it will be added.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := systems[string(namespaceValue)]; ok {
					//Validate that the data contains the mid label with value and check it exists in our temp structure if not it will be added
					if midValue, ok := result.(model.Matrix)[i].Metric[mid]; ok {
						if _, ok := systems[string(namespaceValue)].pointers[prefix+"__"+string(midValue)]; ok {
							//loop through all the labels for an entity and store them in a map. For controller based entities where there will be multiple copies of containers they will have there values concatinated together.
							for key, value := range result.(model.Matrix)[i].Metric {
								addToLabelMap(string(key), string(value), systems[string(namespaceValue)].pointers[prefix+"__"+string(midValue)].labelMap)
							}
						}
					}
				}
			}
		}
	}
}

//getmidMetricString is used to parse the label based results from Prometheus related to mid Entities and store them in the systems data structure.
func getHPAMetricString(result model.Value, namespace model.LabelName, hpa model.LabelName, metric string, clusterName string, promAddr string) {
	hpas := map[string]map[string]string{}
	//Validate there is data in the results.
	if result != nil {
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our temp structure if not it will be added.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := systems[string(namespaceValue)]; ok {
					//Validate that the data contains the mid label with value and check it exists in our temp structure if not it will be added
					if hpaValue, ok := result.(model.Matrix)[i].Metric[hpa]; ok {
						if _, ok := systems[string(namespaceValue)].pointers["Deployment__"+string(hpaValue)]; ok {
							for key, value := range result.(model.Matrix)[i].Metric {
								addToLabelMap(string(key), string(value), systems[string(namespaceValue)].pointers["Deployment__"+string(hpaValue)].labelMap)
							}
						} else if _, ok := systems[string(namespaceValue)].pointers["ReplicaSet__"+string(hpaValue)]; ok {
							for key, value := range result.(model.Matrix)[i].Metric {
								addToLabelMap(string(key), string(value), systems[string(namespaceValue)].pointers["ReplicaSet__"+string(hpaValue)].labelMap)
							}
						} else if _, ok := systems[string(namespaceValue)].pointers["ReplicationController__"+string(hpaValue)]; ok {
							for key, value := range result.(model.Matrix)[i].Metric {
								addToLabelMap(string(key), string(value), systems[string(namespaceValue)].pointers["ReplicationController__"+string(hpaValue)].labelMap)
							}
						} else {
							hpas[string(hpaValue)] = map[string]string{}
							for key, value := range result.(model.Matrix)[i].Metric {
								addToLabelMap(string(key), string(value), hpas[string(hpaValue)])
							}
						}
					}
				}
			}
		}
	}
	if len(hpas) > 0 {
		writeHPAAttributes(clusterName, promAddr, hpas)
	}
}

//getNamespaceMetric is used to parse the results from Prometheus related to Namespace Entities and store them in the systems data structure.
func getNamespacelimits(result model.Value, namespace model.LabelName) {
	//Validate there is data in the results.
	if result != nil {
		//Loop through the different entities in the results.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our systems structure.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := systems[string(namespaceValue)]; ok {
					//validates that the value of the entity is set and if not will default to 0
					var value int
					if len(result.(model.Matrix)[i].Values) == 0 {
						value = 0
					} else {
						value = int(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
					}
					//Check which metric this is for and update the corresponding variable for this container in the system data structure
					//For systems limits they are defined based on 2 of the labels as they combine the Limits and Request for CPU and Memory all into 1 call.

					if constraint := result.(model.Matrix)[i].Metric["constraint"]; constraint == "defaultRequest" {
						if resource := result.(model.Matrix)[i].Metric["resource"]; resource == "cpu" {
							systems[string(namespaceValue)].cpuRequest = value
						} else if resource := result.(model.Matrix)[i].Metric["resource"]; resource == "memory" {
							systems[string(namespaceValue)].memRequest = value
						}
					} else if constraint := result.(model.Matrix)[i].Metric["constraint"]; constraint == "default" {
						if resource := result.(model.Matrix)[i].Metric["resource"]; resource == "cpu" {
							systems[string(namespaceValue)].cpuLimit = value
						} else if resource := result.(model.Matrix)[i].Metric["resource"]; resource == "memory" {
							systems[string(namespaceValue)].memLimit = value
						}
					}
				}
			}
		}
	}
}

//getNamespaceMetricString is used to parse the label based results from Prometheus related to Namespace Entities and store them in the systems data structure.
func getNamespaceMetricString(result model.Value, namespace model.LabelName, metric string) {
	//Validate there is data in the results.
	if result != nil {
		//Loop through the different entities in the results.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our temp structure if not it will be added.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := systems[string(namespaceValue)]; ok {
					//loop through all the labels for an entity and store them in a map.
					for key, value := range result.(model.Matrix)[i].Metric {
						addToLabelMap(string(key), string(value), systems[string(namespaceValue)].labelMap)
					}
				}
			}
		}
	}
}

func getWorkload(promaddress, fileName, metricName, query, aggregator, clusterName, promAddr, interval string, intervalSize, history int, currentTime time.Time) {
	var historyInterval time.Duration
	historyInterval = 0
	var result model.Value
	var start, end time.Time
	//Open the files that will be used for the workload data types and write out there headers.
	workloadWrite, err := os.Create("./data/container/" + aggregator + `_` + fileName + ".csv")
	if err != nil {
		log.Println(err)
	}
	fmt.Fprintf(workloadWrite, "cluster,namespace,top level,top kind,container,Datetime,%s\n", metricName)

	//If the History parameter is set to anything but default 1 then will loop through the calls starting with the current day\hour\minute interval and work backwards.
	//This is done as the farther you go back in time the slpwer prometheus querying becomes and we have seen cases where will not run from timeouts on Prometheus.
	//As a result if we do hit an issue with timing out on Prometheus side we still can send the current data and data going back to that point vs losing it all.
	for historyInterval = 0; int(historyInterval) < history; historyInterval++ {
		start, end = prometheus.TimeRange(interval, intervalSize, currentTime, historyInterval)
		query = aggregator + `(` + query + `) by (pod_name,namespace,container_name)`
		result = prometheus.MetricCollect(promaddress, query, start, end)
		writeWorkload(workloadWrite, result, "namespace", "pod_name", "container_name", clusterName, promAddr)
	}
	//Close the workload files.
	workloadWrite.Close()
}

func getDeploymentWorkload(promaddress, fileName, metricName, query, clusterName, promAddr, interval string, intervalSize, history int, currentTime time.Time) {
	var historyInterval time.Duration
	historyInterval = 0
	var result model.Value
	var start, end time.Time
	//Open the files that will be used for the workload data types and write out there headers.
	workloadWrite, err := os.Create("./data/container/deployment_" + fileName + ".csv")
	if err != nil {
		log.Println(err)
	}
	fmt.Fprintf(workloadWrite, "cluster,namespace,top level,top kind,container,Datetime,%s\n", metricName)

	tempMap := map[int]map[string]map[string][]model.SamplePair{}
	
	for historyInterval = 0; int(historyInterval) < history; historyInterval++ {
		tempMap[int(historyInterval)]=map[string]map[string][]model.SamplePair{}
		start, end = prometheus.TimeRange(interval, intervalSize, currentTime, historyInterval)
		result = prometheus.MetricCollect(promaddress, query, start, end)
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			for j := 0; j < len(result.(model.Matrix)[i].Values); j++ {
				if _,ok := tempMap[int(historyInterval)][string(result.(model.Matrix)[i].Metric["namespace"])]; !ok{
					tempMap[int(historyInterval)][string(result.(model.Matrix)[i].Metric["namespace"])]=map[string][]model.SamplePair{}
				}
				if _,ok := tempMap[int(historyInterval)][string(result.(model.Matrix)[i].Metric["namespace"])][string(result.(model.Matrix)[i].Metric["deployment"])]; !ok {
					tempMap[int(historyInterval)][string(result.(model.Matrix)[i].Metric["namespace"])][string(result.(model.Matrix)[i].Metric["deployment"])] = []model.SamplePair{}
				}
				tempMap[int(historyInterval)][string(result.(model.Matrix)[i].Metric["namespace"])][string(result.(model.Matrix)[i].Metric["deployment"])] = append(tempMap[int(historyInterval)][string(result.(model.Matrix)[i].Metric["namespace"])][string(result.(model.Matrix)[i].Metric["deployment"])], result.(model.Matrix)[i].Values[j])
			}
		}
	}
	var cluster string
	if clusterName == "" {
		cluster = promAddr
	} else {
		cluster = clusterName
	}

	for n := range systems{
		for m, midVal := range systems[n].midLevels{
			if midVal.kind=="Deployment"{
				for c := range systems[n].midLevels[m].containers{
					for historyInterval = 0; int(historyInterval) < history; historyInterval++ {
						for _, val := range tempMap[int(historyInterval)][n][m[12:]] {
							fmt.Fprintf(workloadWrite, "%s,%s,%s,%s,%s,%s,%f\n", cluster, n, midVal.name, midVal.kind, c, time.Unix(0, int64(val.Timestamp)*1000000).Format("2006-01-02 15:04:05.000"), val.Value)
						}
					}
				}
			}
		}
	}
	workloadWrite.Close()
}

func getHPAWorkload(promaddress, fileName, metricName, query, clusterName, promAddr, interval string, intervalSize, history int, currentTime time.Time) {
	var historyInterval time.Duration
	historyInterval = 0
	var result model.Value
	var start, end time.Time
	//Open the files that will be used for the workload data types and write out there headers.
	workloadWrite, err := os.Create("./data/container/hpa" + `_` + fileName + ".csv")
	if err != nil {
		log.Println(err)
	}
	fmt.Fprintf(workloadWrite, "cluster,namespace,deployment,hpa,Datetime,%s\n", metricName)

	//If the History parameter is set to anything but default 1 then will loop through the calls starting with the current day\hour\minute interval and work backwards.
	//This is done as the farther you go back in time the slpwer prometheus querying becomes and we have seen cases where will not run from timeouts on Prometheus.
	//As a result if we do hit an issue with timing out on Prometheus side we still can send the current data and data going back to that point vs losing it all.
	for historyInterval = 0; int(historyInterval) < history; historyInterval++ {
		start, end = prometheus.TimeRange(interval, intervalSize, currentTime, historyInterval)
		result = prometheus.MetricCollect(promaddress, query, start, end)
		writeHPAWorkload(workloadWrite, result, "namespace", "hpa", "hpa", clusterName, promAddr)
	}
	//Close the workload files.
	workloadWrite.Close()
}

func addToLabelMap(key string, value string, labelPath map[string]string) {
	if _, ok := labelPath[key]; !ok {
		labelPath[key] = value
	} else {
		if strings.Contains(value, ";") {
			currValue := ""
			for _, l := range value {
				currValue = currValue + string(l)
				if l == ';' {
					addToLabelMap(key, currValue[:len(currValue)-1], labelPath)
					currValue = ""
				}
			}
			addToLabelMap(key, currValue, labelPath)
		} else {
			currValue := ""
			notPresent := true
			for _, l := range labelPath[key] {
				currValue = currValue + string(l)
				if l == ';' {
					if currValue[:len(currValue)-1] == value {
						notPresent = false
						break
					}
					currValue = ""
				}
			}
			if currValue != value && notPresent {
				labelPath[key] = labelPath[key] + ";" + value
			}
		}
	}
}
