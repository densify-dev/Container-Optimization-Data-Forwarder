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

var clusterName, promAddr, promPort, promProtocol, timeout, aggregator, interval, configFile, configPath string
var intervalSize, history int
var debug bool
var step time.Duration
var currentTime time.Time
var systems = map[string]*Namespace{}

//Container used to hold information related to the containers defined in Kubernetes
type Container struct {
	memory, cpuLimit, cpuRequest, memLimit, memRequest, restarts, powerState int
	conLabel, conInfo, currentNodes, podName                                 string
}

//Pod used to hold information related to the controllers defined or individual pods in Kubernetes
type Pod struct {
	podInfo, podLabel, ownerKind, ownerName string
	currentSize                             int
	creationTime                            int64
	containers                              map[string]*Container
}

//Namespace used to hold information related to the namespaces defined in Kubernetes
type Namespace struct {
	namespaceLabel                             string
	cpuLimit, cpuRequest, memLimit, memRequest int
	pods                                       map[string]*Pod
}

func initParameters() {
	flag.StringVar(&clusterName, "clusterName", "", "Name of the cluster to show in Densify")
	flag.StringVar(&promProtocol, "protocol", "http", "Which protocol to use http|https")
	flag.StringVar(&promAddr, "address", "", "Name of the Prometheus Server")
	flag.StringVar(&promPort, "port", "9090", "Prometheus Port")
	//flag.StringVar(&timeout, "timeout", "3600", "Timeout for querying Prometheus")
	flag.StringVar(&aggregator, "aggregator", "max", "Which aggregator for the data collection of controllers to use max|avg|min")
	flag.StringVar(&interval, "interval", "hours", "Interval to use for data collection. Can be days, hours or minutes")
	flag.IntVar(&intervalSize, "intervalSize", 1, "Interval size to be used for querying. eg. default of 1 with default interval of hours queries 1 last hour of info")
	flag.IntVar(&history, "history", 1, "Amount of time to go back for data collection works with the interval and intervalSize settings")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.StringVar(&configFile, "file", "config", "Name of the config file without extention. Default config")
	flag.StringVar(&configPath, "path", "./config", "Path to where the config file is stored")
	flag.Parse()

	//set defaults
	viper.SetDefault("cluster_name", clusterName)
	viper.SetDefault("prometheus_protocol", promProtocol)
	viper.SetDefault("prometheus_address", promAddr)
	viper.SetDefault("prometheus_port", promPort)
	//viper.SetDefault("timeout", "3600")
	viper.SetDefault("aggregator", aggregator)
	viper.SetDefault("interval", interval)
	viper.SetDefault("interval_size", intervalSize)
	viper.SetDefault("history", history)
	viper.SetDefault("debug", debug)
	// Config import setup.
	viper.SetConfigName(configFile)
	viper.AddConfigPath(configPath)
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s ", err))
	}

	clusterName = viper.GetString("cluster_name")
	promProtocol = viper.GetString("prometheus_protocol")
	promAddr = viper.GetString("prometheus_address")
	promPort = viper.GetString("prometheus_port")
	//timeout = viper.GetString("timeout")
	aggregator = viper.GetString("aggregator")
	interval = viper.GetString("interval")
	intervalSize = viper.GetInt("interval_size")
	history = viper.GetInt("history")
	debug = viper.GetBool("debug")

}

//metricCollect is used to query Prometheus to get data for specific query and return the results to be processed.
func metricCollect(query string, historyInterval time.Duration) (value model.Value) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := api.NewClient(api.Config{Address: promProtocol + "://" + promAddr + ":" + promPort})
	if err != nil {
		log.Fatalln(err)
	}

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
	q := v1.NewAPI(client)
	value, err = q.QueryRange(ctx, query, v1.Range{Start: start, End: end, Step: step})
	//value, err = q.Query(ctx, "kube_pod_container_info", time.Now())
	if err != nil {
		log.Println(err)
	}
	return value
}

func getContainerMetric(result model.Value, namespace, pod, container model.LabelName, metric string) {
	if result != nil {
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := systems[string(namespaceValue)]; ok {
					if podValue, ok := result.(model.Matrix)[i].Metric[pod]; ok {
						if _, ok := systems[string(namespaceValue)].pods[string(podValue)]; ok {
							if containerValue, ok := result.(model.Matrix)[i].Metric[container]; ok {
								if _, ok := systems[string(namespaceValue)].pods[string(podValue)].containers[string(containerValue)]; ok {
									var value int
									if len(result.(model.Matrix)[i].Values) == 0 {
										value = 0
									} else {
										value = int(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
									}
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

func getContainerMetricString(result model.Value, namespace, pod, container model.LabelName, metric string) {
	var tempSystems = map[string]map[string]map[string]map[string]string{}
	if result != nil {
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := systems[string(namespaceValue)]; ok {
					if _, ok := tempSystems[string(namespaceValue)]; ok == false {
						tempSystems[string(namespaceValue)] = map[string]map[string]map[string]string{}
					}
					if podValue, ok := result.(model.Matrix)[i].Metric[pod]; ok {
						if _, ok := systems[string(namespaceValue)].pods[string(podValue)]; ok {
							if _, ok := tempSystems[string(namespaceValue)][string(podValue)]; ok == false {
								tempSystems[string(namespaceValue)][string(podValue)] = map[string]map[string]string{}
							}
							if containerValue, ok := result.(model.Matrix)[i].Metric[container]; ok {
								if _, ok := systems[string(namespaceValue)].pods[string(podValue)].containers[string(containerValue)]; ok {
									if _, ok := tempSystems[string(namespaceValue)][string(podValue)][string(containerValue)]; ok == false {
										tempSystems[string(namespaceValue)][string(podValue)][string(containerValue)] = map[string]string{}
									}
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
		for kn := range tempSystems {
			for kp := range tempSystems[kn] {
				for kc := range tempSystems[kn][kp] {
					tempAttr := ""
					for key, value := range tempSystems[kn][kp][kc] {
						if len(key) < 250 {
							if len(value)+3+len(key) < 256 {
								tempAttr += key + " : " + value + "|"
							} else {
								templength := 256 - 3 - len(key)
								tempAttr += key + " : " + value[:templength] + "|"
							}

						}
						if metric == "conLabel" && key == "instance" {
							systems[kn].pods[kp].containers[kc].currentNodes += strings.Replace(value, ";", "|", -1) + "|"
						} else if metric == "conInfo" && key == "pod" {
							systems[kn].pods[kp].containers[kc].podName = value

						}
					}
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

func getPodMetric(result model.Value, namespace, pod model.LabelName, metric string) {
	if result != nil {
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := systems[string(namespaceValue)]; ok {
					if podValue, ok := result.(model.Matrix)[i].Metric[pod]; ok {
						if _, ok := systems[string(namespaceValue)].pods[string(podValue)]; ok {
							var value int64
							if len(result.(model.Matrix)[i].Values) == 0 {
								value = 0
							} else {
								value = int64(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
							}
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

func getPodMetricString(result model.Value, namespace, pod model.LabelName, metric string) {
	var tempSystems = map[string]map[string]map[string]string{}
	if result != nil {
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := systems[string(namespaceValue)]; ok {
					if _, ok := tempSystems[string(namespaceValue)]; ok == false {
						tempSystems[string(namespaceValue)] = map[string]map[string]string{}
					}
					if podValue, ok := result.(model.Matrix)[i].Metric[pod]; ok {
						if _, ok := systems[string(namespaceValue)].pods[string(podValue)]; ok {
							if _, ok := tempSystems[string(namespaceValue)][string(podValue)]; ok == false {
								tempSystems[string(namespaceValue)][string(podValue)] = map[string]string{}
							}
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
		for kn := range tempSystems {
			for kp := range tempSystems[kn] {
				tempAttr := ""
				for key, value := range tempSystems[kn][kp] {
					if len(key) < 250 {
						if len(value)+3+len(key) < 256 {
							tempAttr += key + " : " + value + "|"
						} else {
							templength := 256 - 3 - len(key)
							tempAttr += key + " : " + value[:templength] + "|"
						}
					}
				}
				tempAttr = tempAttr[:len(tempAttr)-1]
				if metric == "podInfo" {
					systems[kn].pods[kp].podInfo = tempAttr
				} else if metric == "podLabel" {
					systems[kn].pods[kp].podLabel = tempAttr
				}
			}
		}
	}
}

func getNamespacelimits(result model.Value, namespace model.LabelName) {
	if result != nil {
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := systems[string(namespaceValue)]; ok {
					var value int
					if len(result.(model.Matrix)[i].Values) == 0 {
						value = 0
					} else {
						value = int(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
					}
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

func getNamespaceMetricString(result model.Value, namespace model.LabelName, metric string) {
	var tempSystems = map[string]map[string]string{}
	if result != nil {
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := systems[string(namespaceValue)]; ok {
					if _, ok := tempSystems[string(namespaceValue)]; ok == false {
						tempSystems[string(namespaceValue)] = map[string]string{}
					}
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
		for kn := range tempSystems {
			tempAttr := ""
			for key, value := range tempSystems[kn] {
				if len(key) < 250 {
					if len(value)+3+len(key) < 256 {
						tempAttr += key + " : " + value + "|"
					} else {
						templength := 256 - 3 - len(key)
						tempAttr += key + " : " + value[:templength] + "|"
					}
				}
			}
			tempAttr = tempAttr[:len(tempAttr)-1]
			if metric == "namespaceLabel" {
				systems[kn].namespaceLabel = tempAttr
			}
		}
	}
}

func writeConfig() {
	configWrite, err := os.Create("./data/config.csv")
	if err != nil {
		log.Println(err)
	}

	fmt.Fprintln(configWrite, "cluster,namespace,pod,container,HW Total Memory,OS Name,HW Manufacturer,HW Model,HW Serial Number")
	var cluster string
	if clusterName == "" {
		cluster = promAddr
	} else {
		cluster = clusterName
	}
	for kn := range systems {
		for kp := range systems[kn].pods {
			for kc, vc := range systems[kn].pods[kp].containers {
				if vc.memory == -1 {
					fmt.Fprintf(configWrite, "%s,%s,%s,%s,,Linux,CONTAINERS,%s,%s\n", cluster, kn, strings.Replace(kp, ";", ".", -1), strings.Replace(kc, ":", ".", -1), kn, kn)
				} else {
					fmt.Fprintf(configWrite, "%s,%s,%s,%s,%d,Linux,CONTAINERS,%s,%s\n", cluster, kn, strings.Replace(kp, ";", ".", -1), strings.Replace(kc, ":", ".", -1), vc.memory, kn, kn)
				}
			}
		}
	}
}

func writeAttributes() {
	attributeWrite, err := os.Create("./data/attributes.csv")
	if err != nil {
		log.Println(err)
	}

	fmt.Fprintln(attributeWrite, "cluster,namespace,pod,container,Virtual Technology,Virtual Domain,Virtual Datacenter,Virtual Cluster,Container Labels,Container Info,Pod Info,Pod Labels,Existing CPU Limit,Existing CPU Request,Existing Memory Limit,Existing Memory Request,Container Name,Current Nodes,Power State,Created By Kind,Created By Name,Current Size,Create Time,Container Restarts,Namespace Labels,Namespace CPU Request,Namespace CPU Limit,Namespace Memory Request,Namespace Memory Limit")

	var cluster string
	if clusterName == "" {
		cluster = promAddr
	} else {
		cluster = clusterName
	}
	for kn, vn := range systems {
		for kp, vp := range systems[kn].pods {
			for kc, vc := range systems[kn].pods[kp].containers {
				var cstate string
				if vc.powerState == 1 {
					cstate = "Terminated"
				} else {
					cstate = "Running"
				}
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
				fmt.Fprintf(attributeWrite, "\n")
			}
		}
	}
}

func writeWorkload(file io.Writer, result model.Value, namespace, pod, container model.LabelName) {
	if result != nil {
		var cluster string
		if clusterName == "" {
			cluster = promAddr
		} else {
			cluster = clusterName
		}
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := systems[string(namespaceValue)]; ok {
					if podValue, ok := result.(model.Matrix)[i].Metric[pod]; ok {
						if _, ok := systems[string(namespaceValue)].pods[string(podValue)]; ok {
							if containerValue, ok := result.(model.Matrix)[i].Metric[container]; ok {
								if _, ok := systems[string(namespaceValue)].pods[string(podValue)].containers[string(containerValue)]; ok {
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

func main() {

	debugLog, err := os.OpenFile("./data/log.txt", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer debugLog.Close()
	log.SetOutput(debugLog)

	log.Println("Version 1.0.0")

	initParameters()

	var t time.Time
	t = time.Now().UTC()
	if interval == "days" {
		currentTime = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	} else if interval == "hours" {
		currentTime = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
	} else {
		currentTime = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, t.Location())
	}
	//currentTime = time.Now().UTC()

	var historyInterval time.Duration
	historyInterval = 0
	step = 300000000000

	var query string
	var result model.Value

	query = `max(sum(container_spec_memory_limit_bytes{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)/1024/1024 * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind!="<none>"}, "pod_name", "$1", "pod", "(.*)")) by (owner_name,owner_kind,namespace,container_name)`
	result = metricCollect(query, historyInterval)

	if result != nil {
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			if namespaceValue, ok := result.(model.Matrix)[i].Metric["namespace"]; ok {
				if _, ok := systems[string(namespaceValue)]; ok == false {
					systems[string(namespaceValue)] = &Namespace{namespaceLabel: "", cpuRequest: -1, cpuLimit: -1, memRequest: -1, memLimit: -1, pods: map[string]*Pod{}}
				}
				if ownerValue, ok := result.(model.Matrix)[i].Metric["owner_name"]; ok {
					if _, ok := systems[string(namespaceValue)].pods[string(ownerValue)]; ok == false {
						if ownerKind, ok := result.(model.Matrix)[i].Metric["owner_kind"]; ok {
							systems[string(namespaceValue)].pods[string(ownerValue)] = &Pod{podInfo: "", podLabel: "", ownerKind: string(ownerKind), ownerName: string(ownerValue), creationTime: -1, currentSize: -1, containers: map[string]*Container{}}
						} else {
							systems[string(namespaceValue)].pods[string(ownerValue)] = &Pod{podInfo: "", podLabel: "", ownerKind: "", ownerName: string(ownerValue), creationTime: -1, currentSize: -1, containers: map[string]*Container{}}
						}
					}
					if containerValue, ok := result.(model.Matrix)[i].Metric["container_name"]; ok {
						if _, ok := systems[string(namespaceValue)].pods[string(ownerValue)].containers[string(containerValue)]; ok == false {
							var memSize int
							//Check the length of the values array if it is empty then set memory to 0 otherwise use the last\current value in the array as the size of the memory.
							if len(result.(model.Matrix)[i].Values) == 0 {
								memSize = 0
							} else {
								memSize = int(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
							}
							systems[string(namespaceValue)].pods[string(ownerValue)].containers[string(containerValue)] = &Container{memory: memSize, cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1, restarts: -1, powerState: 1, conLabel: "", conInfo: "", currentNodes: "", podName: ""}
						}
					}
				}
			}
		}
	}

	if debug {
		log.Println("[DEBUG] Output of systems after initial call to setup controllers")
		log.Println("[DEBUG] ", systems)
	}

	query = `max(sum(container_spec_memory_limit_bytes{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)/1024/1024 * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind="<none>"}, "pod_name", "$1", "pod", "(.*)")) by (pod_name,namespace,container_name)`
	result = metricCollect(query, historyInterval)

	if result != nil {
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			if namespaceValue, ok := result.(model.Matrix)[i].Metric["namespace"]; ok {
				if _, ok := systems[string(namespaceValue)]; ok == false {
					systems[string(namespaceValue)] = &Namespace{namespaceLabel: "", cpuRequest: -1, cpuLimit: -1, memRequest: -1, memLimit: -1, pods: map[string]*Pod{}}
				}
				if podValue, ok := result.(model.Matrix)[i].Metric["pod_name"]; ok {
					if _, ok := systems[string(namespaceValue)].pods[string(podValue)]; ok == false {
						systems[string(namespaceValue)].pods[string(podValue)] = &Pod{podInfo: "", podLabel: "", ownerKind: "<none>", ownerName: string(podValue), creationTime: -1, currentSize: -1, containers: map[string]*Container{}}
					}
					if containerValue, ok := result.(model.Matrix)[i].Metric["container_name"]; ok {
						if _, ok := systems[string(namespaceValue)].pods[string(podValue)].containers[string(containerValue)]; ok == false {
							var memSize int
							//Check the length of the values array if it is empty then set memory to 0 otherwise use the last\current value in the array as the size of the memory.
							if len(result.(model.Matrix)[i].Values) == 0 {
								memSize = 0
							} else {
								memSize = int(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
							}
							systems[string(namespaceValue)].pods[string(podValue)].containers[string(containerValue)] = &Container{memory: memSize, cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1, restarts: -1, powerState: 1, conLabel: "", conInfo: "", currentNodes: "", podName: ""}
						}
					}
				}
			}
		}
	}
	if len(systems) == 0 {
		fmt.Println("No data returned from Prometheus. Validate all the prerequisites are setup")
		log.Fatalln("No data returned from Prometheus. Validate all the prerequisites are setup")
	}

	if debug {
		log.Println("[DEBUG] Output of systems after initial call to setup individual pods")
		log.Println("[DEBUG] ", systems)
	}
	//var kubeStateOwner, kubeStatePod string
	//kubeStateOwner = ` * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind!="<none>"}) by (owner_name,owner_kind,namespace,container)`
	//kubeStateOwner = ` * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind="<none>"}) by (pod,namespace,container)`

	//Container metrics
	query = `max(sum(kube_pod_container_resource_limits_cpu_cores) by (pod,namespace,container)*1000 * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind!="<none>"}) by (owner_name,owner_kind,namespace,container)`
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "owner_name", "container", "cpuLimit")

	query = `max(sum(kube_pod_container_resource_limits_cpu_cores) by (pod,namespace,container)*1000 * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind="<none>"}) by (pod,namespace,container)`
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "pod", "container", "cpuLimit")

	query = `max(sum(kube_pod_container_resource_requests_cpu_cores) by (pod,namespace,container)*1000 * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind!="<none>"}) by (owner_name,owner_kind,namespace,container)`
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "owner_name", "container", "cpuRequest")

	query = `max(sum(kube_pod_container_resource_requests_cpu_cores) by (pod,namespace,container)*1000 * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind="<none>"}) by (pod,namespace,container)`
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "pod", "container", "cpuRequest")

	query = `max(sum(kube_pod_container_resource_limits_memory_bytes) by (pod,namespace,container)/1024/1024 * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind!="<none>"}) by (owner_name,owner_kind,namespace,container)`
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "owner_name", "container", "memLimit")

	query = `max(sum(kube_pod_container_resource_limits_memory_bytes) by (pod,namespace,container)/1024/1024 * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind="<none>"}) by (pod,namespace,container)`
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "pod", "container", "memLimit")

	query = `max(sum(kube_pod_container_resource_requests_memory_bytes) by (pod,namespace,container)/1024/1024 * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind!="<none>"}) by (owner_name,owner_kind,namespace,container)`
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "owner_name", "container", "memRequest")

	query = `max(sum(kube_pod_container_resource_requests_memory_bytes) by (pod,namespace,container)/1024/1024 * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind="<none>"}) by (pod,namespace,container)`
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "pod", "container", "memRequest")

	query = `max(sum(kube_pod_container_status_restarts_total) by (pod,namespace,container) * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind!="<none>"}) by (owner_name,owner_kind,namespace,container)`
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "owner_name", "container", "restarts")

	query = `max(sum(kube_pod_container_status_restarts_total) by (pod,namespace,container) * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind="<none>"}) by (pod,namespace,container)`
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "pod", "container", "restarts")

	query = `max(sum(kube_pod_container_status_terminated) by (pod,namespace,container) * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind!="<none>"}) by (owner_name,owner_kind,namespace,container)`
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "owner_name", "container", "powerState")

	query = `max(sum(kube_pod_container_status_terminated) by (pod,namespace,container) * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind="<none>"}) by (pod,namespace,container)`
	result = metricCollect(query, historyInterval)
	getContainerMetric(result, "namespace", "pod", "container", "powerState")

	query = `(sum(container_spec_cpu_shares{name!~"k8s_POD_.*"}) by (pod_name,namespace,container_name)) * on (namespace,pod_name,container_name) group_right container_spec_cpu_shares * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind!="<none>"}, "pod_name", "$1", "pod", "(.*)")`
	result = metricCollect(query, historyInterval)
	getContainerMetricString(result, "namespace", "owner_name", "container_name", "conLabel")

	query = `(sum(container_spec_cpu_shares{name!~"k8s_POD_.*"}) by (pod_name,namespace,container_name)) * on (namespace,pod_name,container_name) group_right container_spec_cpu_shares * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind="<none>"}, "pod_name", "$1", "pod", "(.*)")`
	result = metricCollect(query, historyInterval)
	getContainerMetricString(result, "namespace", "pod_name", "container_name", "conLabel")

	query = `sum(kube_pod_container_info) by (pod,namespace,container) * on (namespace,pod,container) group_right kube_pod_container_info * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind!="<none>"}`
	result = metricCollect(query, historyInterval)
	getContainerMetricString(result, "namespace", "owner_name", "container", "conInfo")

	query = `sum(kube_pod_container_info) by (pod,namespace,container) * on (namespace,pod,container) group_right kube_pod_container_info * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind="<none>"}`
	result = metricCollect(query, historyInterval)
	getContainerMetricString(result, "namespace", "pod", "container", "conInfo")

	//Pod Metrics
	query = `sum(kube_pod_container_info) by (pod,namespace) * on (namespace,pod) group_right kube_pod_owner{owner_kind!="<none>"}`
	result = metricCollect(query, historyInterval)
	getPodMetricString(result, "namespace", "owner_name", "podInfo")

	query = `sum(kube_pod_container_info) by (pod,namespace) * on (namespace,pod) group_right kube_pod_owner{owner_kind="<none>"}`
	result = metricCollect(query, historyInterval)
	getPodMetricString(result, "namespace", "pod", "podInfo")

	query = `sum(kube_pod_container_info) by (pod,namespace) * on (namespace,pod) group_right kube_pod_labels * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind!="<none>"}`
	result = metricCollect(query, historyInterval)
	getPodMetricString(result, "namespace", "owner_name", "podLabel")

	query = `sum(kube_pod_container_info) by (pod,namespace) * on (namespace,pod) group_right kube_pod_labels * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind="<none>"}`
	result = metricCollect(query, historyInterval)
	getPodMetricString(result, "namespace", "pod", "podLabel")

	query = `kube_replicaset_spec_replicas`
	result = metricCollect(query, historyInterval)
	getPodMetric(result, "namespace", "replicaset", "currentSize")

	query = `kube_replicationcontroller_spec_replicas`
	result = metricCollect(query, historyInterval)
	getPodMetric(result, "namespace", "replicationcontroller", "currentSize")

	query = `kube_daemonset_status_number_available`
	result = metricCollect(query, historyInterval)
	getPodMetric(result, "namespace", "daemonset", "currentSize")

	query = `kube_statefulset_replicas`
	result = metricCollect(query, historyInterval)
	getPodMetric(result, "namespace", "statefulset", "currentSize")

	query = `kube_job_spec_parallelism`
	result = metricCollect(query, historyInterval)
	getPodMetric(result, "namespace", "job_name", "currentSize")

	query = `max(kube_pod_created * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind!="<none>"}) by (owner_name,owner_kind,namespace,container)`
	result = metricCollect(query, historyInterval)
	getPodMetric(result, "namespace", "owner_name", "creationTime")

	query = `max(kube_pod_created * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind="<none>"}) by (pod,namespace,container)`
	result = metricCollect(query, historyInterval)
	getPodMetric(result, "namespace", "pod", "creationTime")

	//Namespace Metrics
	query = `kube_namespace_labels`
	result = metricCollect(query, historyInterval)
	getNamespaceMetricString(result, "namespace", "namespaceLabel")

	query = `kube_limitrange`
	result = metricCollect(query, historyInterval)
	getNamespacelimits(result, "namespace")

	writeConfig()
	writeAttributes()

	cpuWrite, err := os.Create("./data/cpu_mCores_workload.csv")
	if err != nil {
		log.Println(err)
	}
	fmt.Fprintln(cpuWrite, "cluster,namespace,pod,container,Datetime,CPU Utilization in mCores")
	memWrite, err := os.Create("./data/mem_workload.csv")
	if err != nil {
		log.Println(err)
	}
	fmt.Fprintln(memWrite, "cluster,namespace,pod,container,Datetime,Raw Mem Utilization")
	rssWrite, err := os.Create("./data/rss_workload.csv")
	if err != nil {
		log.Println(err)
	}
	fmt.Fprintln(rssWrite, "cluster,namespace,pod,container,Datetime,Actual Memory Utilization")
	diskWrite, err := os.Create("./data/disk_workload.csv")
	if err != nil {
		log.Println(err)
	}
	fmt.Fprintln(diskWrite, "cluster,namespace,pod,container,Datetime,Raw Disk Utilization")

	//Insert looping logic here.....
	for historyInterval = 0; int(historyInterval) < history; historyInterval++ {
		query = `max(round(sum(rate(container_cpu_usage_seconds_total{name!~"k8s_POD_.*"}[5m])) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)*1000,1) * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind!="<none>"}, "pod_name", "$1", "pod", "(.*)")) by (owner_name,owner_kind,namespace,container_name)`
		result = metricCollect(query, historyInterval)
		writeWorkload(cpuWrite, result, "namespace", "owner_name", "container_name")
		query = `max(round(sum(rate(container_cpu_usage_seconds_total{name!~"k8s_POD_.*"}[5m])) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)*1000,1) * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind="<none>"}, "pod_name", "$1", "pod", "(.*)")) by (pod_name,namespace,container_name)`
		result = metricCollect(query, historyInterval)
		writeWorkload(cpuWrite, result, "namespace", "pod_name", "container_name")

		query = `max(sum(container_memory_usage_bytes{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind) * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind!="<none>"}, "pod_name", "$1", "pod", "(.*)")) by (owner_name,owner_kind,namespace,container_name)`
		result = metricCollect(query, historyInterval)
		writeWorkload(memWrite, result, "namespace", "owner_name", "container_name")
		query = `max(sum(container_memory_usage_bytes{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind) * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind="<none>"}, "pod_name", "$1", "pod", "(.*)")) by (pod_name,namespace,container_name)`
		result = metricCollect(query, historyInterval)
		writeWorkload(memWrite, result, "namespace", "pod_name", "container_name")

		query = `max(sum(container_memory_rss{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind) * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind!="<none>"}, "pod_name", "$1", "pod", "(.*)")) by (owner_name,owner_kind,namespace,container_name)`
		result = metricCollect(query, historyInterval)
		writeWorkload(rssWrite, result, "namespace", "owner_name", "container_name")
		query = `max(sum(container_memory_rss{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind) * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind="<none>"}, "pod_name", "$1", "pod", "(.*)")) by (pod_name,namespace,container_name)`
		result = metricCollect(query, historyInterval)
		writeWorkload(rssWrite, result, "namespace", "pod_name", "container_name")

		query = `max(sum(container_fs_usage_bytes{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind) * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind!="<none>"}, "pod_name", "$1", "pod", "(.*)")) by (owner_name,owner_kind,namespace,container_name)`
		result = metricCollect(query, historyInterval)
		writeWorkload(diskWrite, result, "namespace", "owner_name", "container_name")
		query = `max(sum(container_fs_usage_bytes{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind) * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind="<none>"}, "pod_name", "$1", "pod", "(.*)")) by (pod_name,namespace,container_name)`
		result = metricCollect(query, historyInterval)
		writeWorkload(diskWrite, result, "namespace", "pod_name", "container_name")
	}
}
