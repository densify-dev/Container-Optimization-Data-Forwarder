//Package datacollection collects data from Prometheus and formats the data into CSVs that will be sent to Densify through the Forwarder.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/prometheus/common/model"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/spf13/viper"
)

//Global variables used for Storing system info, command line\config file parameters.
var clusterName, promAddr, promPort, promProtocol, interval, configFile, configPath string
var intervalSize, history int
var debug bool
var step time.Duration
var currentTime time.Time
var systems = map[string]*namespace{}

//container is used to hold information related to the containers defined in a pod
type container struct {
	memory, cpuLimit, cpuRequest, memLimit, memRequest, restarts, powerState int
	conLabel, conInfo, currentNodes, podName                                 string
}

//pod is used to hold information related to the controllers or individual pods in a namespace.
type pod struct {
	podInfo, podLabel, ownerKind, ownerName, controllerLabel string
	currentSize                                              int
	creationTime                                             int64
	containers                                               map[string]*container
}

//namespace is used to hold information related to the namespaces defined in Kubernetes
type namespace struct {
	namespaceLabel                             string
	cpuLimit, cpuRequest, memLimit, memRequest int
	pods                                       map[string]*pod
}

//initParamters will look for settings defined on the command line or in config.properties file and update accordingly. Also defines the default values for these variables.
//Note if the value is defined both on the command line and in the config.properties the value in the config.properties will be used.
func initParameters() {
	//Get the settings passed in from the command line and update the variables as required.
	flag.StringVar(&clusterName, "clusterName", "", "Name of the cluster to show in Densify")
	flag.StringVar(&promProtocol, "protocol", "http", "Which protocol to use http|https")
	flag.StringVar(&promAddr, "address", "", "Name of the Prometheus Server")
	flag.StringVar(&promPort, "port", "9090", "Prometheus Port")
	flag.StringVar(&interval, "interval", "hours", "Interval to use for data collection. Can be days, hours or minutes")
	flag.IntVar(&intervalSize, "intervalSize", 1, "Interval size to be used for querying. eg. default of 1 with default interval of hours queries 1 last hour of info")
	flag.IntVar(&history, "history", 1, "Amount of time to go back for data collection works with the interval and intervalSize settings")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.StringVar(&configFile, "file", "config", "Name of the config file without extention. Default config")
	flag.StringVar(&configPath, "path", "./config", "Path to where the config file is stored")
	flag.Parse()

	//Set defaults for viper to use if setting not found in the config.properties file.
	viper.SetDefault("cluster_name", clusterName)
	viper.SetDefault("prometheus_protocol", promProtocol)
	viper.SetDefault("prometheus_address", promAddr)
	viper.SetDefault("prometheus_port", promPort)
	viper.SetDefault("interval", interval)
	viper.SetDefault("interval_size", intervalSize)
	viper.SetDefault("history", history)
	viper.SetDefault("debug", debug)
	// Config import setup.
	viper.SetConfigName(configFile)
	viper.AddConfigPath(configPath)
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s ", err))
	}

	//Process the config.properties file update the variables as required.
	clusterName = viper.GetString("cluster_name")
	promProtocol = viper.GetString("prometheus_protocol")
	promAddr = viper.GetString("prometheus_address")
	promPort = viper.GetString("prometheus_port")
	interval = viper.GetString("interval")
	intervalSize = viper.GetInt("interval_size")
	history = viper.GetInt("history")
	debug = viper.GetBool("debug")

}

//metricCollect is used to query Prometheus to get data for specific query and return the results to be processed.
func metricCollect(query string, historyInterval time.Duration) (value model.Value) {
	//setup the context to use for the API calls
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//Setup the API client connection
	client, err := api.NewClient(api.Config{Address: promProtocol + "://" + promAddr + ":" + promPort})
	if err != nil {
		log.Fatalln(err)
	}

	//define the start and end times to be used for querying prometheus based on the time the script called.
	//Depending on the Interval and interval size will determine the start and end times.
	//For workload metrics the historyInterval will be set depending on how far back in history we are querying currently. Note it will be 0 for all queries that are not workload related.
	var start, end time.Time
	if interval == "days" {
		start = currentTime.Add(time.Hour * -24 * time.Duration(intervalSize)).Add(time.Hour * -24 * time.Duration(intervalSize) * historyInterval)
		end = currentTime.Add(time.Hour * -24 * time.Duration(intervalSize) * historyInterval)
	} else if interval == "hours" {
		start = currentTime.Add(time.Hour * -1 * time.Duration(intervalSize)).Add(time.Hour * -1 * time.Duration(intervalSize) * historyInterval)
		end = currentTime.Add(time.Hour * -1 * time.Duration(intervalSize) * historyInterval)
	} else {
		start = currentTime.Add(time.Minute * -1 * time.Duration(intervalSize)).Add(time.Minute * -1 * time.Duration(intervalSize) * historyInterval)
		end = currentTime.Add(time.Minute * -1 * time.Duration(intervalSize) * historyInterval)
	}
	//Query prometheus with the values defined above as well as the query that was passed into the function.
	q := v1.NewAPI(client)
	value, err = q.QueryRange(ctx, query, v1.Range{Start: start, End: end, Step: step})
	if err != nil {
		log.Println(err)
	}
	//Return the data that was received from Prometheus.
	return value
}

//getContainerMetric is used to parse the results from Prometheus related to Container Entities and store them in the systems data structure.
func getContainerMetric(result model.Value, namespace, pod, container model.LabelName, metric string) {
	//Validate there is data in the results.
	if result != nil {
		//Loop through the different entities in the results.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our systems structure.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := systems[string(namespaceValue)]; ok {
					//Validate that the data contains the pod label with value and check it exists in our systems structure
					if podValue, ok := result.(model.Matrix)[i].Metric[pod]; ok {
						if _, ok := systems[string(namespaceValue)].pods[string(podValue)]; ok {
							//Validate that the data contains the container label with value and check it exists in our systems structure
							if containerValue, ok := result.(model.Matrix)[i].Metric[container]; ok {
								if _, ok := systems[string(namespaceValue)].pods[string(podValue)].containers[string(containerValue)]; ok {
									//validates that the value of the entity is set and if not will default to 0
									var value int
									if len(result.(model.Matrix)[i].Values) == 0 {
										value = 0
									} else {
										value = int(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
									}
									//Check which metric this is for and update the corresponding variable for this container in the system data structure
									if metric == "cpuLimit" {
										systems[string(namespaceValue)].pods[string(podValue)].containers[string(containerValue)].cpuLimit = value
									} else if metric == "cpuRequest" {
										systems[string(namespaceValue)].pods[string(podValue)].containers[string(containerValue)].cpuRequest = value
									} else if metric == "memLimit" {
										systems[string(namespaceValue)].pods[string(podValue)].containers[string(containerValue)].memLimit = value
									} else if metric == "memRequest" {
										systems[string(namespaceValue)].pods[string(podValue)].containers[string(containerValue)].memRequest = value
									} else if metric == "restarts" {
										systems[string(namespaceValue)].pods[string(podValue)].containers[string(containerValue)].restarts = value
									} else if metric == "powerState" {
										systems[string(namespaceValue)].pods[string(podValue)].containers[string(containerValue)].powerState = value
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
func getContainerMetricString(result model.Value, namespace, pod, container model.LabelName, metric string) {
	//temp structure used to store data while working with it. As we are combining the labels into a formatted string for loading.
	var tempSystems = map[string]map[string]map[string]map[string]string{}
	//Validate there is data in the results.
	if result != nil {
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our temp structure if not it will be added.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := systems[string(namespaceValue)]; ok {
					if _, ok := tempSystems[string(namespaceValue)]; ok == false {
						tempSystems[string(namespaceValue)] = map[string]map[string]map[string]string{}
					}
					//Validate that the data contains the pod label with value and check it exists in our temp structure if not it will be added
					if podValue, ok := result.(model.Matrix)[i].Metric[pod]; ok {
						if _, ok := systems[string(namespaceValue)].pods[string(podValue)]; ok {
							if _, ok := tempSystems[string(namespaceValue)][string(podValue)]; ok == false {
								tempSystems[string(namespaceValue)][string(podValue)] = map[string]map[string]string{}
							}
							//Validate that the data contains the container label with value and check it exists in our temp structure if not it will be added
							if containerValue, ok := result.(model.Matrix)[i].Metric[container]; ok {
								if _, ok := systems[string(namespaceValue)].pods[string(podValue)].containers[string(containerValue)]; ok {
									if _, ok := tempSystems[string(namespaceValue)][string(podValue)][string(containerValue)]; ok == false {
										tempSystems[string(namespaceValue)][string(podValue)][string(containerValue)] = map[string]string{}
									}
									//loop through all the labels for an entity and store them in a map. For controller based entities where there will be multiple copies of containers they will have there values concatinated together.
									for key, value := range result.(model.Matrix)[i].Metric {
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
							systems[kn].pods[kp].containers[kc].currentNodes += strings.Replace(value, ";", "|", -1) + "|"
						} else if metric == "conInfo" && key == "pod" {
							systems[kn].pods[kp].containers[kc].podName = value

						}
					}
					//Write out the combined string into the variable in the systems data structure based on which metric you provided.
					tempAttr = tempAttr[:len(tempAttr)-1]
					if metric == "conInfo" {
						systems[kn].pods[kp].containers[kc].conInfo = tempAttr
					} else if metric == "conLabel" {
						systems[kn].pods[kp].containers[kc].conLabel = tempAttr
					}
				}
			}
		}
	}
}

//getPodMetric is used to parse the results from Prometheus related to Pod Entities and store them in the systems data structure.
func getPodMetric(result model.Value, namespace, pod model.LabelName, metric string) {
	//Validate there is data in the results.
	if result != nil {
		//Loop through the different entities in the results.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our systems structure.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := systems[string(namespaceValue)]; ok {
					//Validate that the data contains the pod label with value and check it exists in our systems structure
					if podValue, ok := result.(model.Matrix)[i].Metric[pod]; ok {
						if _, ok := systems[string(namespaceValue)].pods[string(podValue)]; ok {
							//validates that the value of the entity is set and if not will default to 0
							var value int64
							if len(result.(model.Matrix)[i].Values) == 0 {
								value = 0
							} else {
								value = int64(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
							}
							//Check which metric this is for and update the corresponding variable for this pod in the system data structure
							if metric == "currentSize" {
								systems[string(namespaceValue)].pods[string(podValue)].currentSize = int(value)
							} else if metric == "creationTime" {
								systems[string(namespaceValue)].pods[string(podValue)].creationTime = value
							}
						}
					}
				}
			}
		}
	}
}

//getPodMetricString is used to parse the label based results from Prometheus related to Pod Entities and store them in the systems data structure.
func getPodMetricString(result model.Value, namespace, pod model.LabelName, metric string) {
	var tempSystems = map[string]map[string]map[string]string{}
	//Validate there is data in the results.
	if result != nil {
		//Loop through the different entities in the results.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our temp structure if not it will be added.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := systems[string(namespaceValue)]; ok {
					if _, ok := tempSystems[string(namespaceValue)]; ok == false {
						tempSystems[string(namespaceValue)] = map[string]map[string]string{}
					}
					//Validate that the data contains the pod label with value and check it exists in our temp structure if not it will be added
					if podValue, ok := result.(model.Matrix)[i].Metric[pod]; ok {
						if _, ok := systems[string(namespaceValue)].pods[string(podValue)]; ok {
							if _, ok := tempSystems[string(namespaceValue)][string(podValue)]; ok == false {
								tempSystems[string(namespaceValue)][string(podValue)] = map[string]string{}
							}
							//loop through all the labels for an entity and store them in a map. For controller based entities where there will be multiple copies of containers they will have there values concatinated together.
							for key, value := range result.(model.Matrix)[i].Metric {
								if _, ok := tempSystems[string(namespaceValue)][string(podValue)][string(key)]; ok == false {
									tempSystems[string(namespaceValue)][string(podValue)][string(key)] = strings.Replace(string(value), ",", ";", -1)
								} else {
									if strings.Contains(tempSystems[string(namespaceValue)][string(podValue)][string(key)], strings.Replace(string(value), ",", ";", -1)) {
										tempSystems[string(namespaceValue)][string(podValue)][string(key)] += ";" + strings.Replace(string(value), ",", ";", -1)
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
				tempAttr := ""
				for key, value := range tempSystems[kn][kp] {
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
				if metric == "podInfo" {
					systems[kn].pods[kp].podInfo = tempAttr
				} else if metric == "podLabel" {
					systems[kn].pods[kp].podLabel = tempAttr
				} else if metric == "controllerLabel" {
					systems[kn].pods[kp].controllerLabel = tempAttr
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
				if _, ok := systems[string(namespaceValue)]; ok {
					//validates that the value of the entity is set and if not will default to 0
					var value int
					if len(result.(model.Matrix)[i].Values) == 0 {
						value = 0
					} else {
						value = int(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
					}
					//Check which metric this is for and update the corresponding variable for this container in the system data structure
					//For namespaces limits they are defined based on 2 of the labels as they combine the Limits and Request for CPU and Memory all into 1 call.
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
	var tempSystems = map[string]map[string]string{}
	//Validate there is data in the results.
	if result != nil {
		//Loop through the different entities in the results.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our temp structure if not it will be added.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := systems[string(namespaceValue)]; ok {
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
				systems[kn].namespaceLabel = tempAttr
			}
		}
	}
}

//writeConfig will create the config.csv file that is will be sent Densify by the Forwarder.
func writeConfig() {
	//Create the config file and open it for writing.
	configWrite, err := os.Create("./data/config.csv")
	if err != nil {
		log.Println(err)
	}

	//Write out the header.
	fmt.Fprintln(configWrite, "cluster,namespace,pod,container,HW Total Memory,OS Name,HW Manufacturer,HW Model,HW Serial Number")
	//Check if the cluster parameter is set and if it is then use it for the name of the cluster if not use the prometheus address as the cluster name.
	var cluster string
	if clusterName == "" {
		cluster = promAddr
	} else {
		cluster = clusterName
	}
	//Loop through the systems and write out the config data for each system.
	for kn := range systems {
		for kp := range systems[kn].pods {
			for kc, vc := range systems[kn].pods[kp].containers {
				//If memory is not set then use first write that will leave it blank otherwise use the second that sets the value.
				if vc.memory == -1 {
					fmt.Fprintf(configWrite, "%s,%s,%s,%s,,Linux,CONTAINERS,%s,%s\n", cluster, kn, strings.Replace(kp, ";", ".", -1), strings.Replace(kc, ":", ".", -1), kn, kn)
				} else {
					fmt.Fprintf(configWrite, "%s,%s,%s,%s,%d,Linux,CONTAINERS,%s,%s\n", cluster, kn, strings.Replace(kp, ";", ".", -1), strings.Replace(kc, ":", ".", -1), vc.memory, kn, kn)
				}
			}
		}
	}
}

//writeAttributes will create the attributes.csv file that is will be sent Densify by the Forwarder.
func writeAttributes() {
	//Create the attributes file and open it for writing
	attributeWrite, err := os.Create("./data/attributes.csv")
	if err != nil {
		log.Println(err)
	}

	//Write out the header.
	fmt.Fprintln(attributeWrite, "cluster,namespace,pod,container,Virtual Technology,Virtual Domain,Virtual Datacenter,Virtual Cluster,Container Labels,Container Info,Pod Info,Pod Labels,Existing CPU Limit,Existing CPU Request,Existing Memory Limit,Existing Memory Request,Container Name,Current Nodes,Power State,Created By Kind,Created By Name,Current Size,Create Time,Container Restarts,Namespace Labels,Namespace CPU Request,Namespace CPU Limit,Namespace Memory Request,Namespace Memory Limit,Controller Labels")

	//Check if the cluster parameter is set and if it is then use it for the name of the cluster if not use the prometheus address as the cluster name.
	var cluster string
	if clusterName == "" {
		cluster = promAddr
	} else {
		cluster = clusterName
	}
	//Loop through the systems and write out the attributes data for each system.
	for kn, vn := range systems {
		for kp, vp := range systems[kn].pods {
			for kc, vc := range systems[kn].pods[kp].containers {
				var cstate string
				//convert the powerState from number to string 1 is Terminated and 0 is running.
				if vc.powerState == 1 {
					cstate = "Terminated"
				} else {
					cstate = "Running"
				}
				//Write out the different fields. For fiels that are numeric we don't want to write -1 if it wasn't set so we write a blank if that is the value otherwise we write the number out.
				fmt.Fprintf(attributeWrite, "%s,%s,%s,%s,Containers,%s,%s,%s,%s,%s,%s,%s", cluster, kn, strings.Replace(kp, ";", ".", -1), strings.Replace(kc, ":", ".", -1), cluster, kn, kp, vc.conLabel, vc.conInfo, vp.podInfo, vp.podLabel)
				if vc.cpuLimit == -1 {
					fmt.Fprintf(attributeWrite, ",")
				} else {
					fmt.Fprintf(attributeWrite, ",%d", vc.cpuLimit)
				}
				if vc.cpuLimit == -1 {
					fmt.Fprintf(attributeWrite, ",")
				} else {
					fmt.Fprintf(attributeWrite, ",%d", vc.cpuRequest)
				}
				if vc.memLimit == -1 {
					fmt.Fprintf(attributeWrite, ",")
				} else {
					fmt.Fprintf(attributeWrite, ",%d", vc.memLimit)
				}
				if vc.memRequest == -1 {
					fmt.Fprintf(attributeWrite, ",")
				} else {
					fmt.Fprintf(attributeWrite, ",%d", vc.memRequest)
				}
				fmt.Fprintf(attributeWrite, ",%s,%s,%s,%s,%s", kc, vc.currentNodes[:len(vc.currentNodes)-1], cstate, vp.ownerKind, vp.ownerName)
				if vp.currentSize == -1 {
					fmt.Fprintf(attributeWrite, ",")
				} else {
					fmt.Fprintf(attributeWrite, ",%d", vp.currentSize)
				}
				if vp.creationTime == -1 {
					fmt.Fprintf(attributeWrite, ",")
				} else {
					//Formatting the date into the expexted format. Note the reason for that date is a Go specific way of declaring a format you must use that exact date and time.
					fmt.Fprintf(attributeWrite, ",%s", time.Unix(int64(vp.creationTime), 0).Format("2006-01-02 15:04:05.000"))
				}
				if vc.restarts == -1 {
					fmt.Fprintf(attributeWrite, ",")
				} else {
					fmt.Fprintf(attributeWrite, ",%d", vc.restarts)
				}
				fmt.Fprintf(attributeWrite, ",%s", vn.namespaceLabel)
				if vn.cpuRequest == -1 {
					fmt.Fprintf(attributeWrite, ",")
				} else {
					fmt.Fprintf(attributeWrite, ",%d", vn.cpuRequest)
				}
				if vn.cpuLimit == -1 {
					fmt.Fprintf(attributeWrite, ",")
				} else {
					fmt.Fprintf(attributeWrite, ",%d", vn.cpuLimit)
				}
				if vn.memLimit == -1 {
					fmt.Fprintf(attributeWrite, ",")
				} else {
					fmt.Fprintf(attributeWrite, ",%d", vn.memRequest)
				}
				if vn.memLimit == -1 {
					fmt.Fprintf(attributeWrite, ",")
				} else {
					fmt.Fprintf(attributeWrite, ",%d", vn.memLimit)
				}
				fmt.Fprintf(attributeWrite, ",%s\n", vp.controllerLabel)
			}
		}
	}
}

//writeWorkload will write out the workload data specific to metric provided to the file that was passed in.
func writeWorkload(file io.Writer, result model.Value, namespace, pod, container model.LabelName) {
	if result != nil {
		//Check if the cluster parameter is set and if it is then use it for the name of the cluster if not use the prometheus address as the cluster name.
		var cluster string
		if clusterName == "" {
			cluster = promAddr
		} else {
			cluster = clusterName
		}
		//Loop through the results for the workload and validate that contains the required labels and that the entity exists in the systems data structure once validated will write out the workload for the system.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := systems[string(namespaceValue)]; ok {
					if podValue, ok := result.(model.Matrix)[i].Metric[pod]; ok {
						if _, ok := systems[string(namespaceValue)].pods[string(podValue)]; ok {
							if containerValue, ok := result.(model.Matrix)[i].Metric[container]; ok {
								if _, ok := systems[string(namespaceValue)].pods[string(podValue)].containers[string(containerValue)]; ok {
									//Loop through the different values over the interval and write out each one to the workload file.
									for j := 0; j < len(result.(model.Matrix)[i].Values); j++ {
										fmt.Fprintf(file, "%s,%s,%s,%s,%s,%f\n", cluster, namespaceValue, strings.Replace(string(podValue), ";", ".", -1), strings.Replace(string(containerValue), ":", ".", -1), time.Unix(0, int64(result.(model.Matrix)[i].Values[j].Timestamp)*1000000).Format("2006-01-02 15:04:05.000"), result.(model.Matrix)[i].Values[j].Value)
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

//writeWorkload will write out the workload data specific to metric provided to the file that was passed in.
func writeWorkloadPod(file io.Writer, result model.Value, namespace, pod model.LabelName) {
	if result != nil {
		//Check if the cluster parameter is set and if it is then use it for the name of the cluster if not use the prometheus address as the cluster name.
		var cluster string
		if clusterName == "" {
			cluster = promAddr
		} else {
			cluster = clusterName
		}
		//Loop through the results for the workload and validate that contains the required labels and that the entity exists in the systems data structure once validated will write out the workload for the system.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := systems[string(namespaceValue)]; ok {
					if podValue, ok := result.(model.Matrix)[i].Metric[pod]; ok {
						if _, ok := systems[string(namespaceValue)].pods[string(podValue)]; ok {
							for kc := range systems[string(namespaceValue)].pods[string(podValue)].containers {
								//Loop through the different values over the interval and write out each one to the workload file.
								for j := 0; j < len(result.(model.Matrix)[i].Values); j++ {
									fmt.Fprintf(file, "%s,%s,%s,%s,%s,%f\n", cluster, namespaceValue, strings.Replace(string(podValue), ";", ".", -1), strings.Replace(string(kc), ":", ".", -1), time.Unix(0, int64(result.(model.Matrix)[i].Values[j].Timestamp)*1000000).Format("2006-01-02 15:04:05.000"), result.(model.Matrix)[i].Values[j].Value)
								}
							}

						}
					}
				}
			}
		}
	}
}

func getWorkload(fileName, metricName, query2, aggregrator string) {
	var historyInterval time.Duration
	historyInterval = 0
	var result model.Value
	var query string
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
		//Query for CPU usage in millicores.
		query = aggregrator + `(` + query2 + ` * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind!="<none>"}, "pod_name", "$1", "pod", "(.*)")) by (owner_name,owner_kind,namespace,container_name)`
		result = metricCollect(query, historyInterval)
		writeWorkload(workloadWrite, result, "namespace", "owner_name", "container_name")
		query = aggregrator + `(` + query2 + ` * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind="<none>"}, "pod_name", "$1", "pod", "(.*)")) by (pod_name,namespace,container_name)`
		result = metricCollect(query, historyInterval)
		writeWorkload(workloadWrite, result, "namespace", "pod_name", "container_name")
	}
	//Close the workload files.
	workloadWrite.Close()
}

//main function.
func main() {

	//Open the debug log file for writing.
	debugLog, err := os.OpenFile("./data/log.txt", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer debugLog.Close()
	//Set log to use the debug log for writing output.
	log.SetOutput(debugLog)

	//Version number used for tracking which version of the code the client is using if there is an issue with data collection.
	log.Println("Version 1.0.0")

	//Read in the command line and config file parameters and set the required variables.
	initParameters()

	//Get the current time in UTC and format it. The script uses this time for all the queries this way if you have a large environment we are collecting the data as a snapshot of a specific time and not potentially getting a misaligned set of data.
	var t time.Time
	t = time.Now().UTC()
	if interval == "days" {
		currentTime = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	} else if interval == "hours" {
		currentTime = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
	} else {
		currentTime = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, t.Location())
	}

	//Setup variables used in the code.
	var historyInterval time.Duration
	historyInterval = 0
	//step is set to be 5minutes as it is defined in microseconds.
	step = 300000000000
	var query string
	var result model.Value

	// For the queries we have throughout you will see each query is called twice with minor tweaks this is cause we query once to get all the containers that are part of controllers and a second time to get all the containers that are setup as individual pods.
	//This was done as the label we use for controller based (owner_name) is set to be <none> for all the individual pods and if we query them together for certain fields it would combine values\labels of the individual pods so you would see tags that aren't actually on your container.

	//Query for memory limit set for containers. This query is for the controller based pods.
	query = `max(sum(container_spec_memory_limit_bytes{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)/1024/1024 * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind!="<none>"}, "pod_name", "$1", "pod", "(.*)")) by (owner_name,owner_kind,namespace,container_name)`
	result = metricCollect(query, historyInterval)

	//setup the system data structure for new systems and load existing ones.
	if result != nil {
		//Loop through the different entities in the results.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our systems structure and if not add it.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric["namespace"]; ok {
				if _, ok := systems[string(namespaceValue)]; ok == false {
					systems[string(namespaceValue)] = &namespace{namespaceLabel: "", cpuRequest: -1, cpuLimit: -1, memRequest: -1, memLimit: -1, pods: map[string]*pod{}}
				}
				//Validate that the data contains the owner_name label (This will be the pod field when writing out data) with value and check it exists in our systems structure and if not add it.
				if ownerValue, ok := result.(model.Matrix)[i].Metric["owner_name"]; ok {
					if _, ok := systems[string(namespaceValue)].pods[string(ownerValue)]; ok == false {
						if ownerKind, ok := result.(model.Matrix)[i].Metric["owner_kind"]; ok {
							systems[string(namespaceValue)].pods[string(ownerValue)] = &pod{podInfo: "", podLabel: "", ownerKind: string(ownerKind), ownerName: string(ownerValue), controllerLabel: "", creationTime: -1, currentSize: -1, containers: map[string]*container{}}
						} else {
							systems[string(namespaceValue)].pods[string(ownerValue)] = &pod{podInfo: "", podLabel: "", ownerKind: "", ownerName: string(ownerValue), controllerLabel: "", creationTime: -1, currentSize: -1, containers: map[string]*container{}}
						}
					}
					//Validate that the data contains the container label with value and check it exists in our systems structure and if not add it
					if containerValue, ok := result.(model.Matrix)[i].Metric["container_name"]; ok {
						if _, ok := systems[string(namespaceValue)].pods[string(ownerValue)].containers[string(containerValue)]; ok == false {
							var memSize int
							//Check the length of the values array if it is empty then set memory to 0 otherwise use the last\current value in the array as the size of the memory.
							if len(result.(model.Matrix)[i].Values) == 0 {
								memSize = 0
							} else {
								memSize = int(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
							}
							systems[string(namespaceValue)].pods[string(ownerValue)].containers[string(containerValue)] = &container{memory: memSize, cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1, restarts: -1, powerState: 1, conLabel: "", conInfo: "", currentNodes: "", podName: ""}
						}
					}
				}
			}
		}
	}

	//If debuging on write out the current systems data.
	if debug {
		log.Println("[DEBUG] Output of systems after initial call to setup controllers")
		log.Println("[DEBUG] ", systems)
	}

	//Query for memory limit set for containers. This query is for the individual based pods.
	query = `max(sum(container_spec_memory_limit_bytes{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)/1024/1024 * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind="<none>"}, "pod_name", "$1", "pod", "(.*)")) by (pod_name,namespace,container_name)`
	result = metricCollect(query, historyInterval)

	if result != nil {
		//Loop through the different entities in the results.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our systems structure and if not add it.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric["namespace"]; ok {
				if _, ok := systems[string(namespaceValue)]; ok == false {
					systems[string(namespaceValue)] = &namespace{namespaceLabel: "", cpuRequest: -1, cpuLimit: -1, memRequest: -1, memLimit: -1, pods: map[string]*pod{}}
				}
				//Validate that the data contains the pod_name label (This will be the pod field when writing out data) with value and check it exists in our systems structure and if not add it.
				if podValue, ok := result.(model.Matrix)[i].Metric["pod_name"]; ok {
					if _, ok := systems[string(namespaceValue)].pods[string(podValue)]; ok == false {
						systems[string(namespaceValue)].pods[string(podValue)] = &pod{podInfo: "", podLabel: "", ownerKind: "<none>", ownerName: string(podValue), controllerLabel: "", creationTime: -1, currentSize: -1, containers: map[string]*container{}}
					}
					//Validate that the data contains the container label with value and check it exists in our systems structure and if not add it
					if containerValue, ok := result.(model.Matrix)[i].Metric["container_name"]; ok {
						if _, ok := systems[string(namespaceValue)].pods[string(podValue)].containers[string(containerValue)]; ok == false {
							var memSize int
							//Check the length of the values array if it is empty then set memory to 0 otherwise use the last\current value in the array as the size of the memory.
							if len(result.(model.Matrix)[i].Values) == 0 {
								memSize = 0
							} else {
								memSize = int(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
							}
							systems[string(namespaceValue)].pods[string(podValue)].containers[string(containerValue)] = &container{memory: memSize, cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1, restarts: -1, powerState: 1, conLabel: "", conInfo: "", currentNodes: "", podName: ""}
						}
					}
				}
			}
		}
	}
	//validate there are systems created from 1 of the 2 queries above and if not log error to validate the Prometheus settings and exit.
	if len(systems) == 0 {
		fmt.Println("No data returned from Prometheus. Validate all the prerequisites are setup")
		log.Fatalln("No data returned from Prometheus. Validate all the prerequisites are setup")
	}

	//Write out the systems data structure if debug is enabled.
	if debug {
		log.Println("[DEBUG] Output of systems after initial call to setup individual pods")
		log.Println("[DEBUG] ", systems)
	}

	//variables that were used in prometheus to simplify the repetitive code.
	var kubeStateOwner, kubeStatePod string
	kubeStateOwner = ` * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind!="<none>"}) by (owner_name,owner_kind,namespace,container)`
	kubeStatePod = ` * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind="<none>"}) by (pod,namespace,container)`

	//Container metrics
	//Get the CPU Limit for container
	query = `max(sum(kube_pod_container_resource_limits_cpu_cores) by (pod,namespace,container)*1000` + kubeStateOwner
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "owner_name", "container", "cpuLimit")

	query = `max(sum(kube_pod_container_resource_limits_cpu_cores) by (pod,namespace,container)*1000` + kubeStatePod
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "pod", "container", "cpuLimit")

	//Get the CPU Request for container
	query = `max(sum(kube_pod_container_resource_requests_cpu_cores) by (pod,namespace,container)*1000` + kubeStateOwner
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "owner_name", "container", "cpuRequest")

	query = `max(sum(kube_pod_container_resource_requests_cpu_cores) by (pod,namespace,container)*1000` + kubeStatePod
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "pod", "container", "cpuRequest")

	//Get the Memory Limit for container
	query = `max(sum(kube_pod_container_resource_limits_memory_bytes) by (pod,namespace,container)/1024/1024` + kubeStateOwner
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "owner_name", "container", "memLimit")

	query = `max(sum(kube_pod_container_resource_limits_memory_bytes) by (pod,namespace,container)/1024/1024` + kubeStatePod
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "pod", "container", "memLimit")

	//Get the Memory Request for container
	query = `max(sum(kube_pod_container_resource_requests_memory_bytes) by (pod,namespace,container)/1024/1024` + kubeStateOwner
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "owner_name", "container", "memRequest")

	query = `max(sum(kube_pod_container_resource_requests_memory_bytes) by (pod,namespace,container)/1024/1024` + kubeStatePod
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "pod", "container", "memRequest")

	//Get the number of times the container has been restarted
	query = `max(sum(kube_pod_container_status_restarts_total) by (pod,namespace,container)` + kubeStateOwner
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "owner_name", "container", "restarts")

	query = `max(sum(kube_pod_container_status_restarts_total) by (pod,namespace,container)` + kubeStatePod
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "pod", "container", "restarts")

	//Check to see if the container is still running or if it has been terminated.
	query = `max(sum(kube_pod_container_status_terminated) by (pod,namespace,container)` + kubeStateOwner
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "owner_name", "container", "powerState")

	query = `max(sum(kube_pod_container_status_terminated) by (pod,namespace,container)` + kubeStatePod
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "pod", "container", "powerState")

	//Get the container labels.
	query = `(sum(container_spec_cpu_shares{name!~"k8s_POD_.*"}) by (pod_name,namespace,container_name)) * on (namespace,pod_name,container_name) group_right container_spec_cpu_shares * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind!="<none>"}, "pod_name", "$1", "pod", "(.*)")`
	result = metricCollect(query, historyInterval)
	getContainerMetricString(result, "namespace", "owner_name", "container_name", "conLabel")

	query = `(sum(container_spec_cpu_shares{name!~"k8s_POD_.*"}) by (pod_name,namespace,container_name)) * on (namespace,pod_name,container_name) group_right container_spec_cpu_shares * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind="<none>"}, "pod_name", "$1", "pod", "(.*)")`
	result = metricCollect(query, historyInterval)
	getContainerMetricString(result, "namespace", "pod_name", "container_name", "conLabel")

	//Get the container info values.
	query = `sum(kube_pod_container_info) by (pod,namespace,container) * on (namespace,pod,container) group_right kube_pod_container_info * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind!="<none>"}`
	result = metricCollect(query, historyInterval)
	getContainerMetricString(result, "namespace", "owner_name", "container", "conInfo")

	query = `sum(kube_pod_container_info) by (pod,namespace,container) * on (namespace,pod,container) group_right kube_pod_container_info * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind="<none>"}`
	result = metricCollect(query, historyInterval)
	getContainerMetricString(result, "namespace", "pod", "container", "conInfo")

	//Pod Metrics
	//Get the pod info values
	query = `sum(kube_pod_container_info) by (pod,namespace) * on (namespace,pod) group_right kube_pod_owner{owner_kind!="<none>"}`
	result = metricCollect(query, historyInterval)
	getPodMetricString(result, "namespace", "owner_name", "podInfo")

	query = `sum(kube_pod_container_info) by (pod,namespace) * on (namespace,pod) group_right kube_pod_owner{owner_kind="<none>"}`
	result = metricCollect(query, historyInterval)
	getPodMetricString(result, "namespace", "pod", "podInfo")

	//Get the pod labels.
	query = `sum(kube_pod_container_info) by (pod,namespace) * on (namespace,pod) group_right kube_pod_labels * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind!="<none>"}`
	result = metricCollect(query, historyInterval)
	getPodMetricString(result, "namespace", "owner_name", "podLabel")

	query = `sum(kube_pod_container_info) by (pod,namespace) * on (namespace,pod) group_right kube_pod_labels * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind="<none>"}`
	result = metricCollect(query, historyInterval)
	getPodMetricString(result, "namespace", "pod", "podLabel")

	currentSizeWrite, err := os.Create("./data/currentSize.csv")
	if err != nil {
		log.Println(err)
	}
	fmt.Fprintf(currentSizeWrite, "cluster,namespace,pod,container,Datetime,currentSize\n")

	//Get the current size of the controller will query each of the differnt types of controller
	query = `kube_replicaset_spec_replicas`
	result = metricCollect(query, historyInterval)
	getPodMetric(result, "namespace", "replicaset", "currentSize")
	writeWorkloadPod(currentSizeWrite, result, "namespace", "replicaset")

	query = `kube_replicationcontroller_spec_replicas`
	result = metricCollect(query, historyInterval)
	getPodMetric(result, "namespace", "replicationcontroller", "currentSize")
	writeWorkloadPod(currentSizeWrite, result, "namespace", "replicationcontroller")

	query = `kube_daemonset_status_number_available`
	result = metricCollect(query, historyInterval)
	getPodMetric(result, "namespace", "daemonset", "currentSize")
	writeWorkloadPod(currentSizeWrite, result, "namespace", "daemonset")

	query = `kube_statefulset_replicas`
	result = metricCollect(query, historyInterval)
	getPodMetric(result, "namespace", "statefulset", "currentSize")
	writeWorkloadPod(currentSizeWrite, result, "namespace", "statefulset")

	query = `kube_job_spec_parallelism`
	result = metricCollect(query, historyInterval)
	getPodMetric(result, "namespace", "job_name", "currentSize")
	writeWorkloadPod(currentSizeWrite, result, "namespace", "job_name")

	currentSizeWrite.Close()

	//Get the controller labels
	query = `kube_statefulset_labels`
	result = metricCollect(query, historyInterval)
	getPodMetricString(result, "namespace", "statefulset", "controllerLabel")

	query = `kube_job_labels`
	result = metricCollect(query, historyInterval)
	getPodMetricString(result, "namespace", "job_name", "controllerLabel")

	query = `kube_daemonset_labels`
	result = metricCollect(query, historyInterval)
	getPodMetricString(result, "namespace", "daemonset", "controllerLabel")

	//Get when the pod was originally created.
	query = `max(kube_pod_created` + kubeStateOwner
	result = metricCollect(query, historyInterval)
	getPodMetric(result, "namespace", "owner_name", "creationTime")

	query = `max(kube_pod_created` + kubeStatePod
	result = metricCollect(query, historyInterval)
	getPodMetric(result, "namespace", "pod", "creationTime")

	//Namespace Metrics
	//Get the namespace labels
	query = `kube_namespace_labels`
	result = metricCollect(query, historyInterval)
	getNamespaceMetricString(result, "namespace", "namespaceLabel")

	//Get the CPU and Memory Limit and Request quotes for the namespace.
	query = `kube_limitrange`
	result = metricCollect(query, historyInterval)
	getNamespacelimits(result, "namespace")

	//Write out the config and attributes files.
	writeConfig()
	writeAttributes()

	query = `round(sum(rate(container_cpu_usage_seconds_total{name!~"k8s_POD_.*"}[5m])) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)*1000,1)`
	getWorkload("cpu_mCores_workload", "CPU Utilization in mCores", query, "max")
	query = `sum(container_memory_usage_bytes{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)`
	getWorkload("mem_workload", "Raw Mem Utilization", query, "max")
	query = `sum(container_memory_rss{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)`
	getWorkload("rss_workload", "Actual Memory Utilization", query, "max")
	query = `sum(container_fs_usage_bytes{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)`
	getWorkload("disk_workload", "Raw Disk Utilization", query, "max")

	query = `round(sum(rate(container_cpu_usage_seconds_total{name!~"k8s_POD_.*"}[5m])) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)*1000,1)`
	getWorkload("cpu_mCores_workload", "CPU Utilization in mCores", query, "avg")
	query = `sum(container_memory_usage_bytes{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)`
	getWorkload("mem_workload", "Raw Mem Utilization", query, "avg")
	query = `sum(container_memory_rss{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)`
	getWorkload("rss_workload", "Actual Memory Utilization", query, "avg")
	query = `sum(container_fs_usage_bytes{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)`
	getWorkload("disk_workload", "Raw Disk Utilization", query, "avg")
}
