//Package container2 collects data related to containers and formats into csv files to send to Densify.
package container2

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
	"github.com/prometheus/common/model"
)

//writeConfig will create the config.csv file that is will be sent Densify by the Forwarder.
func writeConfig(clusterName, promAddr string) {
	//Create the config file and open it for writing.
	configWrite, err := os.Create("./data/container/config.csv")
	if err != nil {
		log.Println(prometheus.LogMessage("[ERROR]", promAddr, entityKind, "N/A", err.Error(), "N/A"))
	}

	//Write out the header.
	fmt.Fprintln(configWrite, "cluster,namespace,entity_name,entity_type,container,HW Total Memory,OS Name,HW Manufacturer")
	//Check if the cluster parameter is set and if it is then use it for the name of the cluster if not use the prometheus address as the cluster name.
	var cluster string
	if clusterName == "" {
		cluster = promAddr
	} else {
		cluster = clusterName
	}
	//Loop through the systems and write out the config data for each system.
	for kn := range systems {
		for kt, vt := range systems[kn].midLevels {
			for kc, vc := range systems[kn].midLevels[kt].containers {
				//If memory is not set then use first write that will leave it blank otherwise use the second that sets the value.
				if vc.memory == -1 || vc.memory == 0 {
					fmt.Fprintf(configWrite, "%s,%s,%s,%s,%s,,Linux,CONTAINERS\n", cluster, kn, vt.name, vt.kind, strings.Replace(kc, ":", ".", -1))
				} else {
					fmt.Fprintf(configWrite, "%s,%s,%s,%s,%s,%d,Linux,CONTAINERS\n", cluster, kn, vt.name, vt.kind, strings.Replace(kc, ":", ".", -1), vc.memory)
				}
			}
		}
	}
}

//writeConfig will create the config.csv file that is will be sent Densify by the Forwarder.
func writeHPAConfig(clusterName, promAddr string, systems map[string]map[string]string) {
	//Create the config file and open it for writing.
	configWrite, err := os.Create("./data/hpa/hpa_extra_config.csv")
	if err != nil {
		log.Println(prometheus.LogMessage("[ERROR]", promAddr, entityKind, "N/A", err.Error(), "N/A"))
	}

	//Write out the header.
	fmt.Fprintln(configWrite, "cluster,namespace,entity_name,entity_type,container,HPA Name,OS Name,HW Manufacturer")
	//Check if the cluster parameter is set and if it is then use it for the name of the cluster if not use the prometheus address as the cluster name.
	var cluster string
	if clusterName == "" {
		cluster = promAddr
	} else {
		cluster = clusterName
	}
	//Loop through the systems and write out the config data for each system.
	for i := range systems {
		//Write out the different fields. For fiels that are numeric we don't want to write -1 if it wasn't set so we write a blank if that is the value otherwise we write the number out.
		fmt.Fprintf(configWrite, "%s,%s,,,,%s,Linux,HPA", cluster, systems[i]["namespace"], i)
		fmt.Fprintf(configWrite, "\n")
	}
}

//writeAttributes will create the attributes.csv file that is will be sent Densify by the Forwarder.
func writeAttributes(clusterName, promAddr string) {
	//Create the attributes file and open it for writing
	attributeWrite, err := os.Create("./data/container/attributes.csv")
	if err != nil {
		log.Println(prometheus.LogMessage("[ERROR]", promAddr, entityKind, "N/A", err.Error(), "N/A"))
	}

	//Write out the header.
	fmt.Fprintln(attributeWrite, "cluster,namespace,entity_name,entity_type,container,Virtual Technology,Virtual Domain,Virtual Datacenter,Virtual Cluster,Container Labels,Pod Labels,Existing CPU Limit,Existing CPU Request,Existing Memory Limit,Existing Memory Request,Container Name,Current Nodes,Power State,Created By Kind,Created By Name,Current Size,Create Time,Container Restarts,Namespace Labels,Namespace CPU Request,Namespace CPU Limit,Namespace Memory Request,Namespace Memory Limit")

	//Check if the cluster parameter is set and if it is then use it for the name of the cluster if not use the prometheus address as the cluster name.
	var cluster string
	if clusterName == "" {
		cluster = promAddr
	} else {
		cluster = clusterName
	}
	//Loop through the systems and write out the attributes data for each system.
	for kn, vn := range systems {
		for kt, vt := range systems[kn].midLevels {
			for kc, vc := range systems[kn].midLevels[kt].containers {
				var cstate string
				//convert the powerState from number to string 1 is Terminated and 0 is running.
				if vc.powerState == 1 {
					cstate = "Terminated"
				} else {
					cstate = "Running"
				}

				//for wt, vt := range systems[kn].pods
				//Write out the different fields. For fiels that are numeric we don't want to write -1 if it wasn't set so we write a blank if that is the value otherwise we write the number out.
				fmt.Fprintf(attributeWrite, "%s,%s,%s,%s,%s,Containers,%s,%s,%s,", cluster, kn, strings.Replace(vt.name, ";", ".", -1), vt.kind, strings.Replace(kc, ":", ".", -1), cluster, kn, vt.name)
				for key, value := range systems[kn].midLevels[kt].containers[kc].labelMap {
					if len(key) < 250 {
						value = strings.Replace(value, ",", " ", -1)
						if len(value)+3+len(key) < 256 {
							fmt.Fprintf(attributeWrite, key+" : "+value+"|")
						} else {
							templength := 256 - 3 - len(key)
							fmt.Fprintf(attributeWrite, key+" : "+value[:templength]+"|")
						}
					}
				}
				fmt.Fprintf(attributeWrite, ",")

				for key, value := range vt.labelMap {
					if len(key) < 250 {
						value = strings.Replace(value, ",", " ", -1)
						if len(value)+3+len(key) < 256 {
							fmt.Fprintf(attributeWrite, key+" : "+value+"|")
						} else {
							templength := 256 - 3 - len(key)
							fmt.Fprintf(attributeWrite, key+" : "+value[:templength]+"|")
						}
					}
				}

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
				fmt.Fprintf(attributeWrite, ",%s,%s,%s,%s,%s", kc, strings.Replace(vt.labelMap["node"], ";", "|", -1), cstate, vt.kind, vt.name)
				if vt.currentSize == -1 {
					fmt.Fprintf(attributeWrite, ",")
				} else {
					fmt.Fprintf(attributeWrite, ",%d", vt.currentSize)
				}
				if vt.creationTime == -1 {
					fmt.Fprintf(attributeWrite, ",")
				} else {
					//Formatting the date into the expexted format. Note the reason for that date is a Go specific way of declaring a format you must use that exact date and time.
					fmt.Fprintf(attributeWrite, ",%s", time.Unix(int64(vt.creationTime), 0).Format("2006-01-02 15:04:05.000"))
				}
				if vc.restarts == -1 {
					fmt.Fprintf(attributeWrite, ",")
				} else {
					fmt.Fprintf(attributeWrite, ",%d,", vc.restarts)
				}
				for key, value := range vn.labelMap {
					if len(key) < 250 {
						value = strings.Replace(value, ",", " ", -1)
						if len(value)+3+len(key) < 256 {
							fmt.Fprintf(attributeWrite, key+" : "+value+"|")
						} else {
							templength := 256 - 3 - len(key)
							fmt.Fprintf(attributeWrite, key+" : "+value[:templength]+"|")
						}
					}
				}
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

//writeAttributes will create the attributes.csv file that is will be sent Densify by the Forwarder.
func writeHPAAttributes(clusterName, promAddr string, systems map[string]map[string]string) {
	//Create the attributes file and open it for writing
	attributeWrite, err := os.Create("./data/hpa/hpa_extra_attributes.csv")
	if err != nil {
		log.Println(prometheus.LogMessage("[ERROR]", promAddr, entityKind, "N/A", err.Error(), "N/A"))
	}

	var cluster string
	if clusterName == "" {
		cluster = promAddr
	} else {
		cluster = clusterName
	}

	//Write out the header.
	fmt.Fprintln(attributeWrite, "cluster,namespace,entity_name,entity_type,container,HPA Name,Labels")
	//Loop through the systems and write out the attributes data for each system.
	for i := range systems {
		//Write out the different fields. For fiels that are numeric we don't want to write -1 if it wasn't set so we write a blank if that is the value otherwise we write the number out.
		fmt.Fprintf(attributeWrite, "%s,%s,,,,%s,", cluster, systems[i]["namespace"], i)
		for key, value := range systems[i] {
			value = strings.Replace(value, ",", " ", -1)
			if len(value)+3+len(key) < 256 {
				fmt.Fprintf(attributeWrite, key+" : "+value+"|")
			} else {
				templength := 256 - 3 - len(key)
				fmt.Fprintf(attributeWrite, key+" : "+value[:templength]+"|")
			}
		}
		fmt.Fprintf(attributeWrite, "\n")
	}
}

//writeWorkload will write out the workload data specific to metric provided to the file that was passed in.
func writeWorkload(file io.Writer, result model.Value, namespace, pod, container model.LabelName, clusterName, promAddr string, kind string) {
	var tempKind bool
	if result != nil {
		if kind == "" {
			tempKind = true
		}
		//Check if the cluster parameter is set and if it is then use it for the name of the cluster if not use the prometheus address as the cluster name.
		var cluster string
		if clusterName == "" {
			cluster = promAddr
		} else {
			cluster = clusterName
		}
		//Loop through the results for the workload and validate that contains the required labels and that the entity exists in the systems data structure once validated will write out the workload for the system.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			if tempKind {
				kind = string(result.(model.Matrix)[i].Metric["owner_kind"])
			}
			if namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]; ok {
				if _, ok := systems[string(namespaceValue)]; ok {
					if podValue, ok := result.(model.Matrix)[i].Metric[pod]; ok {
						if _, ok := systems[string(namespaceValue)].midLevels[kind+"__"+string(podValue)]; ok {
							if containerValue, ok := result.(model.Matrix)[i].Metric[container]; ok {
								if _, ok := systems[string(namespaceValue)].midLevels[kind+"__"+string(podValue)].containers[string(containerValue)]; ok {
									//Loop through the different values over the interval and write out each one to the workload file.
									for j := 0; j < len(result.(model.Matrix)[i].Values); j++ {
										fmt.Fprintf(file, "%s,%s,%s,%s,%s,%s,%f\n", cluster, namespaceValue, systems[string(namespaceValue)].midLevels[kind+"__"+string(podValue)].name, systems[string(namespaceValue)].midLevels[kind+"__"+string(podValue)].kind, strings.Replace(string(containerValue), ":", ".", -1), time.Unix(0, int64(result.(model.Matrix)[i].Values[j].Timestamp)*1000000).Format("2006-01-02 15:04:05.000"), result.(model.Matrix)[i].Values[j].Value)
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
func writeHPAWorkload(file io.Writer, result model.Value, namespace, hpa, container model.LabelName, clusterName, promAddr string) {
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
					if hpaValue, ok := result.(model.Matrix)[i].Metric[hpa]; ok {
						if _, ok := systems[string(namespaceValue)].pointers["Deployment__"+string(hpaValue)]; ok {
							for j := 0; j < len(result.(model.Matrix)[i].Values); j++ {
								fmt.Fprintf(file, "%s,%s,%s,%s,%s,%f\n", cluster, namespaceValue, strings.Replace(string(hpaValue), ";", ".", -1), strings.Replace(string(hpaValue), ":", ".", -1), time.Unix(0, int64(result.(model.Matrix)[i].Values[j].Timestamp)*1000000).Format("2006-01-02 15:04:05.000"), result.(model.Matrix)[i].Values[j].Value)
							}
						} else if _, ok := systems[string(namespaceValue)].pointers["ReplicaSet__"+string(hpaValue)]; ok {
							for j := 0; j < len(result.(model.Matrix)[i].Values); j++ {
								fmt.Fprintf(file, "%s,%s,%s,%s,%s,%f\n", cluster, namespaceValue, strings.Replace(string(hpaValue), ";", ".", -1), strings.Replace(string(hpaValue), ":", ".", -1), time.Unix(0, int64(result.(model.Matrix)[i].Values[j].Timestamp)*1000000).Format("2006-01-02 15:04:05.000"), result.(model.Matrix)[i].Values[j].Value)
							}
						} else if _, ok := systems[string(namespaceValue)].pointers["ReplicationController__"+string(hpaValue)]; ok {
							for j := 0; j < len(result.(model.Matrix)[i].Values); j++ {
								fmt.Fprintf(file, "%s,%s,%s,%s,%s,%f\n", cluster, namespaceValue, strings.Replace(string(hpaValue), ";", ".", -1), strings.Replace(string(hpaValue), ":", ".", -1), time.Unix(0, int64(result.(model.Matrix)[i].Values[j].Timestamp)*1000000).Format("2006-01-02 15:04:05.000"), result.(model.Matrix)[i].Values[j].Value)
							}
						} else {
							for j := 0; j < len(result.(model.Matrix)[i].Values); j++ {
								fmt.Fprintf(file, "%s,%s,,%s,%s,%f\n", cluster, namespaceValue, strings.Replace(string(hpaValue), ":", ".", -1), time.Unix(0, int64(result.(model.Matrix)[i].Values[j].Timestamp)*1000000).Format("2006-01-02 15:04:05.000"), result.(model.Matrix)[i].Values[j].Value)
							}
						}
					}
				}
			}
		}
	}
}

//writeWorkload will write out the workload data specific to metric provided to the file that was passed in.
func writeWorkloadMid(file io.Writer, result model.Value, namespace, mid model.LabelName, clusterName, promAddr string, prefix string) {
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
					if midValue, ok := result.(model.Matrix)[i].Metric[mid]; ok {
						//fmt.Println(systems[string(namespaceValue)].mids[string(midValue)].midInfo)
						//fmt.Println(namespaceValue, "-------", midValue)
						//fmt.Println(result.(model.Matrix)[i].Metric[mid])
						if _, ok := systems[string(namespaceValue)].pointers[prefix+"__"+string(midValue)]; ok { //NOT PASSING THIS STATMENT
							for kc := range systems[string(namespaceValue)].pointers[prefix+"__"+string(midValue)].containers {
								//Loop through the different values over the interval and write out each one to the workload file.
								for j := 0; j < len(result.(model.Matrix)[i].Values); j++ {
									fmt.Fprintf(file, "%s,%s,%s,%s,%s,%s,%f\n", cluster, namespaceValue, systems[string(namespaceValue)].pointers[prefix+"__"+string(midValue)].name, systems[string(namespaceValue)].pointers[prefix+"__"+string(midValue)].kind, strings.Replace(string(kc), ":", ".", -1), time.Unix(0, int64(result.(model.Matrix)[i].Values[j].Timestamp)*1000000).Format("2006-01-02 15:04:05.000"), result.(model.Matrix)[i].Values[j].Value)
								}
							}
						}
					}
				}
			}
		}
	}
}
