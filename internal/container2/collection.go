//Package container2 collects data related to containers and formats into csv files to send to Densify.
package container2

import (
	//"fmt"
	//"log"
	//"os"
	"strings"
	//"time"

	//"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
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
				if _, ok := namespaces[string(namespaceValue)]; ok {
					//Validate that the data contains the pod label with value and check it exists in our systems structure
					if podValue, ok := result.(model.Matrix)[i].Metric[pod]; ok {
						if _, ok := namespaces[string(namespaceValue)].pointers["Pod__"+string(podValue)]; ok {
							//Validate that the data contains the container label with value and check it exists in our systems structure
							if containerValue, ok := result.(model.Matrix)[i].Metric[container]; ok {
								if _, ok := namespaces[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)]; ok {
									//validates that the value of the entity is set and if not will default to 0
									var value int
									if len(result.(model.Matrix)[i].Values) == 0 {
										value = 0
									} else {
										value = int(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
									}
									//Check which metric this is for and update the corresponding variable for this container in the system data structure
									if metric == "cpuLimit" {
										namespaces[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].cpuLimit = value
									} else if metric == "cpuRequest" {
										namespaces[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].cpuRequest = value
									} else if metric == "memLimit" {
										namespaces[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].memLimit = value
									} else if metric == "memRequest" {
										namespaces[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].memRequest = value
									} else if metric == "restarts" {
										namespaces[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].restarts = value
									} else if metric == "powerState" {
										namespaces[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].powerState = value
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
	//temp structure used to store data while working with it. As we are combining the labels into a formatted string for loading.
	var tempSystems = map[string]map[string]map[string]map[string]string{}
	//Validate there is data in the results.
	if result != nil {
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our temp structure if not it will be added.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := namespaces[string(namespaceValue)]; ok {
					if _, ok := tempSystems[string(namespaceValue)]; ok == false {
						tempSystems[string(namespaceValue)] = map[string]map[string]map[string]string{}
					}
					//Validate that the data contains the pod label with value and check it exists in our temp structure if not it will be added
					if podValue, ok := result.(model.Matrix)[i].Metric[pod]; ok {
						if _, ok := namespaces[string(namespaceValue)].pointers["Pod__"+string(podValue)]; ok {
							if _, ok := tempSystems[string(namespaceValue)][string(podValue)]; ok == false {
								tempSystems[string(namespaceValue)][string(podValue)] = map[string]map[string]string{}
							}
							//Validate that the data contains the container label with value and check it exists in our temp structure if not it will be added
							if containerValue, ok := result.(model.Matrix)[i].Metric[container]; ok {
								if _, ok := namespaces[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)]; ok {
									if _, ok := tempSystems[string(namespaceValue)][string(podValue)][string(containerValue)]; ok == false {
										tempSystems[string(namespaceValue)][string(podValue)][string(containerValue)] = map[string]string{}
									}
									//loop through all the labels for an entity and store them in a map. For controller based entities where there will be multiple copies of containers they will have there values concatinated together.
									for key, value := range result.(model.Matrix)[i].Metric {
										addToLabelMap(string(key), string(value), namespaces[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].labelMap)
										if _, ok := tempSystems[string(namespaceValue)][string(podValue)][string(containerValue)][string(key)]; ok == false {
											tempSystems[string(namespaceValue)][string(podValue)][string(containerValue)][string(key)] = strings.Replace(string(value), ",", ";", -1)
										} else {
											if strings.Contains(tempSystems[string(namespaceValue)][string(podValue)][string(containerValue)][string(key)], strings.Replace(string(value), ",", ";", -1)) {
												tempSystems[string(namespaceValue)][string(podValue)][string(containerValue)][string(key)] += ";" + strings.Replace(string(value), ",", ";", -1)
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
		//Process the temp data structure to produce 1 string that will written into specific variable in the system data structure.
		for kn := range tempSystems {
			for kp := range tempSystems[kn] {
				for kc := range tempSystems[kn][kp] {
					tempAttr := ""
					for key, value := range tempSystems[kn][kp][kc] {
						//Validate the length of the key and value to be less then 256 characters when combined together per value in the attribute to be loaded.
						if len(key) < 250 {
							if len(value)+3+len(key) < 256 {
								tempAttr += key + " : " + value + "|"
							} else {
								templength := 256 - 3 - len(key)
								tempAttr += key + " : " + value[:templength] + "|"
							}

						}
						// If the label (key) is one a few specific values and the metric is match defined then store the value in an additional location in the systems data structure.
						if metric == "conLabel" && key == "instance" {
							//fmt.Println(strings.Replace(value, ";", "|", -1) + "|")
							//namespaces[kn].pointers[kp].containers[kc].currentNodes += strings.Replace(value, ";", "|", -1) + "|"
						} else if metric == "conInfo" && key == "pod" {
							//namespaces[kn].pointers[kp].containers[kc].podName += strings.Replace(value, ";", "|", -1) + "|"

						}
					}
					//Write out the combined string into the variable in the systems data structure based on which metric you provided.
					tempAttr = tempAttr[:len(tempAttr)-1]
					if metric == "conInfo" {
						//namespaces[kn].pointers[kp].containers[kc].conInfo = tempAttr
					} else if metric == "conLabel" {
						//namespaces[kn].pointers[kp].containers[kc].conLabel = tempAttr
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
				if _, ok := namespaces[string(namespaceValue)]; ok {
					//Validate that the data contains the mid label with value and check it exists in our systems structure
					if midValue, ok := result.(model.Matrix)[i].Metric[mid]; ok {
						if _, ok := namespaces[string(namespaceValue)].pointers[prefix+"__"+string(midValue)]; ok {
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
								namespaces[string(namespaceValue)].pointers[prefix+"__"+string(midValue)].currentSize = int(value)
							} else if metric == "creationTime" {
								//namespaces[string(namespaceValue)].pointers[prefix+"__"+string(midValue)].creationTime = value
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
	var tempSystems = map[string]map[string]map[string]string{}
	//Validate there is data in the results.
	if result != nil {
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our temp structure if not it will be added.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := namespaces[string(namespaceValue)]; ok {
					if _, ok := tempSystems[string(namespaceValue)]; ok == false {
						tempSystems[string(namespaceValue)] = map[string]map[string]string{}
					}
					//Validate that the data contains the mid label with value and check it exists in our temp structure if not it will be added
					if midValue, ok := result.(model.Matrix)[i].Metric[mid]; ok {
						if _, ok := namespaces[string(namespaceValue)].pointers[prefix+"__"+string(midValue)]; ok {
							if _, ok := tempSystems[string(namespaceValue)][string(midValue)]; ok == false {
								tempSystems[string(namespaceValue)][string(midValue)] = map[string]string{}
							}

							//loop through all the labels for an entity and store them in a map. For controller based entities where there will be multiple copies of containers they will have there values concatinated together.
							for key, value := range result.(model.Matrix)[i].Metric {
								addToLabelMap(string(key), string(value), namespaces[string(namespaceValue)].pointers[prefix+"__"+string(midValue)].labelMap)
							}
						}
					}
				}
			}
		}
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
				if _, ok := namespaces[string(namespaceValue)]; ok {
					//validates that the value of the entity is set and if not will default to 0
					var value int
					if len(result.(model.Matrix)[i].Values) == 0 {
						value = 0
					} else {
						value = int(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
					}
					_ = value
					//Check which metric this is for and update the corresponding variable for this container in the system data structure
					//For namespaces limits they are defined based on 2 of the labels as they combine the Limits and Request for CPU and Memory all into 1 call.
					
						if constraint := result.(model.Matrix)[i].Metric["constraint"]; constraint == "defaultRequest" {
							if resource := result.(model.Matrix)[i].Metric["resource"]; resource == "cpu" {
								namespaces[string(namespaceValue)].cpuRequest = value
							} else if resource := result.(model.Matrix)[i].Metric["resource"]; resource == "memory" {
								namespaces[string(namespaceValue)].memRequest = value
							}
						} else if constraint := result.(model.Matrix)[i].Metric["constraint"]; constraint == "default" {
							if resource := result.(model.Matrix)[i].Metric["resource"]; resource == "cpu" {
								namespaces[string(namespaceValue)].cpuLimit = value
							} else if resource := result.(model.Matrix)[i].Metric["resource"]; resource == "memory" {
								namespaces[string(namespaceValue)].memLimit = value
							}
						}
				}
			}
		}
	}
}

//getNamespaceMetricString is used to parse the label based results from Prometheus related to Namespace Entities and store them in the systems data structure.
func getNamespaceMetricString(result model.Value, namespace model.LabelName, metric string) {
	var tempSystems = map[string]map[string]string{}
	//Validate there is data in the results.
	if result != nil {
		//Loop through the different entities in the results.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our temp structure if not it will be added.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := namespaces[string(namespaceValue)]; ok {
					if _, ok := tempSystems[string(namespaceValue)]; ok == false {
						tempSystems[string(namespaceValue)] = map[string]string{}
					}
					//loop through all the labels for an entity and store them in a map.
					for key, value := range result.(model.Matrix)[i].Metric {
						if _, ok := tempSystems[string(namespaceValue)][string(key)]; ok == false {
							tempSystems[string(namespaceValue)][string(key)] = strings.Replace(string(value), ",", ";", -1)
						} else {
							if strings.Contains(tempSystems[string(namespaceValue)][string(key)], strings.Replace(string(value), ",", ";", -1)) {
								tempSystems[string(namespaceValue)][string(key)] += ";" + strings.Replace(string(value), ",", ";", -1)
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
				if metric == "namespaceLabel" {
					namespaces[kn].namespaceLabel = tempAttr
				}
		}
	}
}
/*
func getWorkload(promaddress, fileName, metricName, query, aggregrator, clusterName, promAddr, interval string, intervalSize, history int, currentTime time.Time) {
	var historyInterval time.Duration
	historyInterval = 0
	var result model.Value
	var start, end time.Time
	//Open the files that will be used for the workload data types and write out there headers.
	workloadWrite, err := os.Create("./data/" + aggregrator + `_` + fileName + ".csv")
	if err != nil {
		log.Println(err)
	}
	fmt.Fprintf(workloadWrite, "cluster,namespace,pod,container,Datetime,%s\n", metricName)

	//If the History parameter is set to anything but default 1 then will loop through the calls starting with the current day\hour\minute interval and work backwards.
	//This is done as the farther you go back in time the slpwer prometheus querying becomes and we have seen cases where will not run from timeouts on Prometheus.
	//As a result if we do hit an issue with timing out on Prometheus side we still can send the current data and data going back to that point vs losing it all.
	for historyInterval = 0; int(historyInterval) < history; historyInterval++ {
		start, end = prometheus.TimeRange(interval, intervalSize, currentTime, historyInterval)
		query = aggregrator + `(` + query + `) by (pod_name,namespace,container_name)`
		result = prometheus.MetricCollect(promaddress, query, start, end)
		writeWorkload(workloadWrite, result, "namespace", "pod_name", "container_name", clusterName, promAddr)
	}
	//Close the workload files.
	workloadWrite.Close()
}*/

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