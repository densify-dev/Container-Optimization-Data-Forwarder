//Package hpa collects data related to containers and formats into csv files to send to Densify.
package hpa

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
	"github.com/prometheus/common/model"

	"strings"
)

//getHPAMetric is used to parse the results from Prometheus related to HPA Entities and store them in the systems data structure.
func getHPAMetric(result model.Value, namespace, hpa model.LabelName, metric string) {
	//Validate there is data in the results.
	if result != nil {
		//Loop through the different entities in the results.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our systems structure.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := namespaces[string(namespaceValue)]; ok {
					//Validate that the data contains the hpa label with value and check it exists in our systems structure
					if hpaValue, ok := result.(model.Matrix)[i].Metric[hpa]; ok {
						if _, ok := namespaces[string(namespaceValue)].hpas[string(hpaValue)]; ok {
							//validates that the value of the entity is set and if not will default to 0
							var value int64
							if len(result.(model.Matrix)[i].Values) == 0 {
								value = 0
							} else {
								value = int64(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
							}
							//Check which metric this is for and update the corresponding variable for this hpa in the system data structure
							if metric == "maxReplicas" {
								namespaces[string(namespaceValue)].hpas[string(hpaValue)].maxReplicas = int(value)
							} else if metric == "minReplicas" {
								namespaces[string(namespaceValue)].hpas[string(hpaValue)].minReplicas = int(value)
							}
						}
					}
				}
			}
		}
	}
}

//getHPAMetricString is used to parse the label based results from Prometheus related to HPA Entities and store them in the systems data structure.
func getHPAMetricString(result model.Value, namespace, hpa model.LabelName, metric string) {
	var tempSystems = map[string]map[string]map[string]string{}
	//Validate there is data in the results.
	if result != nil {
		//Loop through the different entities in the results.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our temp structure if not it will be added.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := namespaces[string(namespaceValue)]; ok {
					if _, ok := tempSystems[string(namespaceValue)]; ok == false {
						tempSystems[string(namespaceValue)] = map[string]map[string]string{}
					}
					//Validate that the data contains the hpa label with value and check it exists in our temp structure if not it will be added
					if hpaValue, ok := result.(model.Matrix)[i].Metric[hpa]; ok {
						if _, ok := namespaces[string(namespaceValue)].hpas[string(hpaValue)]; ok {
							if _, ok := tempSystems[string(namespaceValue)][string(hpaValue)]; ok == false {
								tempSystems[string(namespaceValue)][string(hpaValue)] = map[string]string{}
							}
							//loop through all the labels for an entity and store them in a map. For controller based entities where there will be multiple copies of containers they will have there values concatinated together.
							for key, value := range result.(model.Matrix)[i].Metric {
								if _, ok := tempSystems[string(namespaceValue)][string(hpaValue)][string(key)]; ok == false {
									tempSystems[string(namespaceValue)][string(hpaValue)][string(key)] = strings.Replace(string(value), ",", ";", -1)
								} else {
									if strings.Contains(tempSystems[string(namespaceValue)][string(hpaValue)][string(key)], strings.Replace(string(value), ",", ";", -1)) {
										tempSystems[string(namespaceValue)][string(hpaValue)][string(key)] += ";" + strings.Replace(string(value), ",", ";", -1)
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
			for kd := range tempSystems[kn] {
				tempAttr := ""
				for key, value := range tempSystems[kn][kd] {
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
				if metric == "hpaLabel" {
					namespaces[kn].hpas[kd].hpaLabel = tempAttr
				}
			}
		}
	}
}

func getWorkload(promaddress, fileName, metricName, query, clusterName, promAddr, interval string, intervalSize, history int, currentTime time.Time) {
	var historyInterval time.Duration
	historyInterval = 0
	var result model.Value
	var start, end time.Time
	//Open the files that will be used for the workload data types and write out there headers.
	workloadWrite, err := os.Create("./data/hpa/" + fileName + ".csv")
	if err != nil {
		log.Println(err)
	}
	fmt.Fprintf(workloadWrite, "cluster,namespace,hpa,Datetime,%s\n", metricName)

	//If the History parameter is set to anything but default 1 then will loop through the calls starting with the current day\hour\minute interval and work backwards.
	//This is done as the farther you go back in time the slpwer prometheus querying becomes and we have seen cases where will not run from timeouts on Prometheus.
	//As a result if we do hit an issue with timing out on Prometheus side we still can send the current data and data going back to that point vs losing it all.
	for historyInterval = 0; int(historyInterval) < history; historyInterval++ {
		start, end = prometheus.TimeRange(interval, intervalSize, currentTime, historyInterval)
		//Query for CPU usage in millicores.
		result = prometheus.MetricCollect(promaddress, query, start, end)
		writeWorkload(workloadWrite, result, "namespace", "container_name", clusterName, promAddr)
	}
	//Close the workload files.
	workloadWrite.Close()
}
