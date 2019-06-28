//Package cronjob collects data related to jobs and formats into csv files to send to Densify.
package cronjob

import (
	"github.com/prometheus/common/model"

	"strings"
)

//getJobMetric is used to parse the results from Prometheus related to Job Entities and store them in the namespaces data structure.
func getJobMetric(result model.Value, namespace, cronjob, job model.LabelName, metric string) {
	//Validate there is data in the results.
	if result != nil {
		//Loop through the different entities in the results.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our namespaces structure.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := namespaces[string(namespaceValue)]; ok {
					//Validate that the data contains the cronjob label with value and check it exists in our namespaces structure
					if cronjobValue, ok := result.(model.Matrix)[i].Metric[cronjob]; ok {
						if _, ok := namespaces[string(namespaceValue)].cronjobs[string(cronjobValue)]; ok {
							//Validate that the data contains the job label with value and check it exists in our namespaces structure
							if jobValue, ok := result.(model.Matrix)[i].Metric[job]; ok {
								if _, ok := namespaces[string(namespaceValue)].cronjobs[string(cronjobValue)].jobs[string(jobValue)]; ok {
									//validates that the value of the entity is set and if not will default to 0
									var value int
									if len(result.(model.Matrix)[i].Values) == 0 {
										value = 0
									} else {
										value = int(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
									}
									//Check which metric this is for and update the corresponding variable for this job in the system data structure
									if metric == "specCompletions" {
										namespaces[string(namespaceValue)].cronjobs[string(cronjobValue)].jobs[string(jobValue)].specCompletions = value
									} else if metric == "specParallelism" {
										namespaces[string(namespaceValue)].cronjobs[string(cronjobValue)].jobs[string(jobValue)].specParallelism = value
									} else if metric == "statusActive" {
										namespaces[string(namespaceValue)].cronjobs[string(cronjobValue)].jobs[string(jobValue)].statusActive = value
									} else if metric == "statusFailed" {
										namespaces[string(namespaceValue)].cronjobs[string(cronjobValue)].jobs[string(jobValue)].statusFailed = value
									} else if metric == "statusSucceeded" {
										namespaces[string(namespaceValue)].cronjobs[string(cronjobValue)].jobs[string(jobValue)].statusSucceeded = value
									} else if metric == "statusCompletionTime" {
										namespaces[string(namespaceValue)].cronjobs[string(cronjobValue)].jobs[string(jobValue)].statusCompletionTime = value
									} else if metric == "statusStartTime" {
										namespaces[string(namespaceValue)].cronjobs[string(cronjobValue)].jobs[string(jobValue)].statusStartTime = value
									} else if metric == "complete" {
										namespaces[string(namespaceValue)].cronjobs[string(cronjobValue)].jobs[string(jobValue)].complete = value
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

//getJobMetricString is used to parse the label based results from Prometheus related to Job Entities and store them in the namespaces data structure.
func getJobMetricString(result model.Value, namespace, cronjob, job model.LabelName, metric string) {
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
					//Validate that the data contains the cronjob label with value and check it exists in our temp structure if not it will be added
					if cronjobValue, ok := result.(model.Matrix)[i].Metric[cronjob]; ok {
						if _, ok := namespaces[string(namespaceValue)].cronjobs[string(cronjobValue)]; ok {
							if _, ok := tempSystems[string(namespaceValue)][string(cronjobValue)]; ok == false {
								tempSystems[string(namespaceValue)][string(cronjobValue)] = map[string]map[string]string{}
							}
							//Validate that the data contains the job label with value and check it exists in our temp structure if not it will be added
							if jobValue, ok := result.(model.Matrix)[i].Metric[job]; ok {
								//fmt.Println(jobValue, "JOB IS OKAY")
								if _, ok := namespaces[string(namespaceValue)].cronjobs[string(cronjobValue)].jobs[string(jobValue)]; ok {
									if _, ok := tempSystems[string(namespaceValue)][string(cronjobValue)][string(jobValue)]; ok == false {
										tempSystems[string(namespaceValue)][string(cronjobValue)][string(jobValue)] = map[string]string{}
									}
									//loop through all the labels for an entity and store them in a map. For controller based entities where there will be multiple copies of jobs they will have there values concatinated together.
									for key, value := range result.(model.Matrix)[i].Metric {
										if _, ok := tempSystems[string(namespaceValue)][string(cronjobValue)][string(jobValue)][string(key)]; ok == false {
											tempSystems[string(namespaceValue)][string(cronjobValue)][string(jobValue)][string(key)] = strings.Replace(string(value), ",", ";", -1)
										} else {
											if strings.Contains(tempSystems[string(namespaceValue)][string(cronjobValue)][string(jobValue)][string(key)], strings.Replace(string(value), ",", ";", -1)) {
												tempSystems[string(namespaceValue)][string(cronjobValue)][string(jobValue)][string(key)] += ";" + strings.Replace(string(value), ",", ";", -1)
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
		//fmt.Println(tempSystems["default"]["densify-job"]["densify-job-1558958400"]["instance"])
		for kn := range tempSystems {
			for kc := range tempSystems[kn] {
				for kj := range tempSystems[kn][kc] {
					//fmt.Println(v)
					tempAttr := ""
					for key, value := range tempSystems[kn][kc][kj] {
						//fmt.Println(value)
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
					//Write out the combined string into the variable in the namespaces data structure based on which metric you provided.
					tempAttr = tempAttr[:len(tempAttr)-1]
					if metric == "jobInfo" {
						namespaces[kn].cronjobs[kc].jobs[kj].jobInfo = tempAttr
					} else if metric == "jobLabel" {
						namespaces[kn].cronjobs[kc].jobs[kj].jobLabel = tempAttr
					}
					//fmt.Println(namespaces[kn].cronjobs[kc].jobs[kj].jobLabel)
				}
			}
		}
	}
}

//getCronJobMetric is used to parse the results from Prometheus related to CronJob Entities and store them in the namespaces data structure.
func getCronJobMetric(result model.Value, namespace, cronjob model.LabelName, metric string) {
	//Validate there is data in the results.
	if result != nil {
		//Loop through the different entities in the results.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our namespaces structure.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := namespaces[string(namespaceValue)]; ok {
					//Validate that the data contains the cronjob label with value and check it exists in our namespaces structure
					if cronjobValue, ok := result.(model.Matrix)[i].Metric[cronjob]; ok {
						if _, ok := namespaces[string(namespaceValue)].cronjobs[string(cronjobValue)]; ok {
							//validates that the value of the entity is set and if not will default to 0
							var value int64
							if len(result.(model.Matrix)[i].Values) == 0 {
								value = 0
							} else {
								value = int64(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
							}
							//Check which metric this is for and update the corresponding variable for this cronjob in the system data structure
							if metric == "statusActive" {
								namespaces[string(namespaceValue)].cronjobs[string(cronjobValue)].statusActive = int(value)
							} else if metric == "nextScheduleTime" {
								namespaces[string(namespaceValue)].cronjobs[string(cronjobValue)].nextScheduleTime = int(value)
							} else if metric == "lastScheduleTime" {
								namespaces[string(namespaceValue)].cronjobs[string(cronjobValue)].lastScheduleTime = int(value)
							}
						}
					}
				}
			}
		}
	}
}

//getCronJobMetricString is used to parse the label based results from Prometheus related to CronJob Entities and store them in the namespaces data structure.
func getCronJobMetricString(result model.Value, namespace, cronjob model.LabelName, metric string) {
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
					//Validate that the data contains the cronjob label with value and check it exists in our temp structure if not it will be added
					if cronjobValue, ok := result.(model.Matrix)[i].Metric[cronjob]; ok {
						if _, ok := namespaces[string(namespaceValue)].cronjobs[string(cronjobValue)]; ok {
							if _, ok := tempSystems[string(namespaceValue)][string(cronjobValue)]; ok == false {
								tempSystems[string(namespaceValue)][string(cronjobValue)] = map[string]string{}
							}
							//loop through all the labels for an entity and store them in a map. For controller based entities where there will be multiple copies of jobs they will have there values concatinated together.
							for key, value := range result.(model.Matrix)[i].Metric {
								if _, ok := tempSystems[string(namespaceValue)][string(cronjobValue)][string(key)]; ok == false {
									tempSystems[string(namespaceValue)][string(cronjobValue)][string(key)] = strings.Replace(string(value), ",", ";", -1)
								} else {
									if strings.Contains(tempSystems[string(namespaceValue)][string(cronjobValue)][string(key)], strings.Replace(string(value), ",", ";", -1)) {
										tempSystems[string(namespaceValue)][string(cronjobValue)][string(key)] += ";" + strings.Replace(string(value), ",", ";", -1)
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
			for kc := range tempSystems[kn] {
				tempAttr := ""
				for key, value := range tempSystems[kn][kc] {
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
				//Write out the combined string into the variable in the namespaces data structure based on which metric you provided.
				tempAttr = tempAttr[:len(tempAttr)-1]
				if metric == "cronjobLabel" {
					namespaces[kn].cronjobs[kc].cronjobLabel = tempAttr
				}
			}
		}
	}
}

/*
func getWorkload(promaddress, fileName, metricName, query, clusterName, promAddr, interval string, intervalSize, history int, currentTime time.Time) {
	var historyInterval time.Duration
	historyInterval = 0
	var result model.Value
	var start, end time.Time
	//Open the files that will be used for the workload data types and write out there headers.
	workloadWrite, err := os.Create("./data/cronjob/" + fileName + ".csv")
	if err != nil {
		log.Println(err)
	}
	fmt.Fprintf(workloadWrite, "cluster,namespace,cronjob,Datetime,%s\n", metricName)

	//If the History parameter is set to anything but default 1 then will loop through the calls starting with the current day\hour\minute interval and work backwards.
	//This is done as the farther you go back in time the slpwer prometheus querying becomes and we have seen cases where will not run from timeouts on Prometheus.
	//As a result if we do hit an issue with timing out on Prometheus side we still can send the current data and data going back to that point vs losing it all.
	for historyInterval = 0; int(historyInterval) < history; historyInterval++ {
		start, end = prometheus.TimeRange(interval, intervalSize, currentTime, historyInterval)
		//Query for CPU usage in millicores.
		result = prometheus.MetricCollect(promaddress, query, start, end)
		writeWorkload(workloadWrite, result, "namespace", "job_name", clusterName, promAddr)
	}
	//Close the workload files.
	workloadWrite.Close()
}*/
