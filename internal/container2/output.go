//Package container2 collects data related to containers and formats into csv files to send to Densify.
package container2

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/prometheus/common/model"
)

//writeConfig will create the config.csv file that is will be sent Densify by the Forwarder.
func writeConfig(clusterName, promAddr string) {
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
	for kn := range namespaces {
		for kt := range namespaces[kn].topLevels {
			for kc, vc := range namespaces[kn].topLevels[kt].containers {
				//If memory is not set then use first write that will leave it blank otherwise use the second that sets the value.
				if vc.memory == -1 {
					fmt.Fprintf(configWrite, "%s,%s,%s,%s,,Linux,CONTAINERS,%s,%s\n", cluster, kn, strings.Replace(kt, ";", ".", -1), strings.Replace(kc, ":", ".", -1), kn, kn)
				} else {
					fmt.Fprintf(configWrite, "%s,%s,%s,%s,%d,Linux,CONTAINERS,%s,%s\n", cluster, kn, strings.Replace(kt, ";", ".", -1), strings.Replace(kc, ":", ".", -1), vc.memory, kn, kn)
				}
			}
		}
	}
}

//writeAttributes will create the attributes.csv file that is will be sent Densify by the Forwarder.
func writeAttributes(clusterName, promAddr string) {
	//Create the attributes file and open it for writing
	attributeWrite, err := os.Create("./data/attributes.csv")
	if err != nil {
		log.Println(err)
	}

	//Write out the header.
	fmt.Fprintln(attributeWrite, "cluster,namespace,top level,top kind,container,pod name,Virtual Technology,Virtual Domain,Virtual Datacenter,Virtual Cluster,All Container Labels,All Mid Level Labels,Existing CPU Limit,Existing CPU Request,Existing Memory Limit,Existing Memory Request,Container Name,Current Nodes,Power State,Created By Kind,Created By Name,Current Size,Create Time,Container Restarts,Namespace Labels,Namespace CPU Request,Namespace CPU Limit,Namespace Memory Request,Namespace Memory Limit")

	//Check if the cluster parameter is set and if it is then use it for the name of the cluster if not use the prometheus address as the cluster name.
	var cluster string
	if clusterName == "" {
		cluster = promAddr
	} else {
		cluster = clusterName
	}
	//Loop through the systems and write out the attributes data for each system.
	for kn, vn := range namespaces {
		for kt, vt := range namespaces[kn].topLevels {
			for kc, vc := range namespaces[kn].topLevels[kt].containers {
				var cstate string
				//convert the powerState from number to string 1 is Terminated and 0 is running.
				if vc.powerState == 1 {
					cstate = "Terminated"
				} else {
					cstate = "Running"
				}

				//for wt, vt := range namespaces[kn].pods
				//Write out the different fields. For fiels that are numeric we don't want to write -1 if it wasn't set so we write a blank if that is the value otherwise we write the number out.
				fmt.Fprintf(attributeWrite, "%s,%s,%s,%s,%s,%s,Containers,%s,%s,%s,", cluster, kn, strings.Replace(vt.name, ";", ".", -1), vt.kind, strings.Replace(kc, ":", ".", -1), vc.labelMap["pod"], cluster, kn, vt.name)
				for key, value := range namespaces[kn].topLevels[kt].containers[kc].labelMap {
					fmt.Fprintf(attributeWrite, key+" : "+value+"|")
				}
				fmt.Fprintf(attributeWrite, ",")

				for key, value := range vt.labelMap {
					fmt.Fprintf(attributeWrite, key+" : "+value+"|")
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
				fmt.Fprintf(attributeWrite, ",%s,%s,%s,%s,%s", kc, vt.labelMap["node_instance"], cstate, vt.kind, vt.name)
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
					fmt.Fprintf(attributeWrite, key+" : "+value+"|")
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

//writeWorkload will write out the workload data specific to metric provided to the file that was passed in.
func writeWorkload(file io.Writer, result model.Value, namespace, pod, container model.LabelName, clusterName, promAddr string) {
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
				if _, ok := namespaces[string(namespaceValue)]; ok {
					if podValue, ok := result.(model.Matrix)[i].Metric[pod]; ok {
						if _, ok := namespaces[string(namespaceValue)].pointers["Pod__"+string(podValue)]; ok {
							if containerValue, ok := result.(model.Matrix)[i].Metric[container]; ok {
								if _, ok := namespaces[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)]; ok {
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
				if _, ok := namespaces[string(namespaceValue)]; ok {
					if midValue, ok := result.(model.Matrix)[i].Metric[mid]; ok {
						//fmt.Println(namespaces[string(namespaceValue)].mids[string(midValue)].midInfo)
						//fmt.Println(namespaceValue, "-------", midValue)
						//fmt.Println(result.(model.Matrix)[i].Metric[mid])
						if _, ok := namespaces[string(namespaceValue)].pointers[prefix+"__"+string(midValue)]; ok { //NOT PASSING THIS STATMENT
							for kc := range namespaces[string(namespaceValue)].pointers[prefix+"__"+string(midValue)].containers {
								//Loop through the different values over the interval and write out each one to the workload file.
								for j := 0; j < len(result.(model.Matrix)[i].Values); j++ {
									fmt.Fprintf(file, "%s,%s,%s,%s,%s,%f\n", cluster, namespaceValue, strings.Replace(string(midValue), ";", ".", -1), strings.Replace(string(kc), ":", ".", -1), time.Unix(0, int64(result.(model.Matrix)[i].Values[j].Timestamp)*1000000).Format("2006-01-02 15:04:05.000"), result.(model.Matrix)[i].Values[j].Value)
								}
							}
						}
					}
				}
			}
		}
	}
}