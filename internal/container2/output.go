//Package container2 collects data related to containers and formats into csv files to send to Densify.
package container2

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/common"
	"github.com/prometheus/common/model"
)

//writeConfig will create the config.csv file that is will be sent to Densify by the Forwarder.
func writeConfig(args *common.Parameters) {
	//Create the config file and open it for writing.
	configWrite, err := os.Create("./data/container/config.csv")
	if err != nil {
		args.ErrorLogger.Println("entity=" + entityKind + " message=" + err.Error())
		fmt.Println("[ERROR] entity=" + entityKind + " message=" + err.Error())
		return
	}

	//Write out the header.
	fmt.Fprintln(configWrite, "cluster,namespace,entity_name,entity_type,container,HW Total Memory,OS Name,HW Manufacturer")

	//Loop through the systems and write out the config data for each system.
	for kn := range systems {
		for kt, vt := range systems[kn].midLevels {
			for kc, vc := range systems[kn].midLevels[kt].containers {
				//If memory is not set then use first write that will leave it blank otherwise use the second that sets the value.
				if vc.memory == -1 || vc.memory == 0 {
					fmt.Fprintf(configWrite, "%s,%s,%s,%s,%s,,Linux,CONTAINERS\n", *args.ClusterName, kn, vt.name, vt.kind, strings.Replace(kc, ":", ".", -1))
				} else {
					fmt.Fprintf(configWrite, "%s,%s,%s,%s,%s,%d,Linux,CONTAINERS\n", *args.ClusterName, kn, vt.name, vt.kind, strings.Replace(kc, ":", ".", -1), vc.memory)
				}
			}
		}
	}
}

//writeConfig will create the config.csv file that is will be sent to Densify by the Forwarder.
func writeHPAConfig(args *common.Parameters, systems map[string]map[string]string) {
	//Create the config file and open it for writing.
	configWrite, err := os.Create("./data/hpa/hpa_extra_config.csv")
	if err != nil {
		args.ErrorLogger.Println("entity=" + entityKind + " message=" + err.Error())
		fmt.Println("[ERROR] entity=" + entityKind + " message=" + err.Error())
		return
	}

	//Write out the header.
	fmt.Fprintln(configWrite, "cluster,namespace,entity_name,entity_type,container,HPA Name,OS Name,HW Manufacturer")

	//Loop through the systems and write out the config data for each system.
	for i := range systems {
		//Write out the different fields. For fiels that are numeric we don't want to write -1 if it wasn't set so we write a blank if that is the value otherwise we write the number out.
		fmt.Fprintf(configWrite, "%s,%s,,,,%s,Linux,HPA", *args.ClusterName, systems[i]["namespace"], i)
		fmt.Fprintf(configWrite, "\n")
	}
}

//writeAttributes will create the attributes.csv file that is will be sent to Densify by the Forwarder.
func writeAttributes(args *common.Parameters) {
	//Create the attributes file and open it for writing
	attributeWrite, err := os.Create("./data/container/attributes.csv")
	if err != nil {
		args.ErrorLogger.Println("entity=" + entityKind + " message=" + err.Error())
		fmt.Println("[ERROR] entity=" + entityKind + " message=" + err.Error())
		return
	}

	//Write out the header.
	fmt.Fprintln(attributeWrite, "cluster,namespace,entity_name,entity_type,container,Virtual Technology,Virtual Domain,Virtual Datacenter,Virtual Cluster,Container Labels,Pod Labels,Existing CPU Limit,Existing CPU Request,Existing Memory Limit,Existing Memory Request,Container Name,Current Nodes,Power State,Created By Kind,Created By Name,Current Size,Create Time,Container Restarts,Namespace Labels,Namespace CPU Request,Namespace CPU Limit,Namespace Memory Request,Namespace Memory Limit")

	//Loop through the systems and write out the attributes data for each system.
	for kn, vn := range systems {
		for kt, vt := range systems[kn].midLevels {
			for kc, vc := range systems[kn].midLevels[kt].containers {
				var cstate = "Running"
				//convert the powerState from number to string 1 is Terminated and 0 is running.
				if vc.powerState == 1 {
					cstate = "Terminated"
				}
				//for wt, vt := range systems[kn].pods
				//Write out the different fields. For fiels that are numeric we don't want to write -1 if it wasn't set so we write a blank if that is the value otherwise we write the number out.
				fmt.Fprintf(attributeWrite, "%s,%s,%s,%s,%s,Containers,%s,%s,%s,", *args.ClusterName, kn, strings.Replace(vt.name, ";", ".", -1), vt.kind, strings.Replace(kc, ":", ".", -1), *args.ClusterName, kn, vt.name)
				for key, value := range systems[kn].midLevels[kt].containers[kc].labelMap {
					if len(key) >= 250 {
						continue
					}
					value = strings.Replace(value, ",", " ", -1)
					if len(value)+3+len(key) < 256 {
						fmt.Fprintf(attributeWrite, key+" : "+value+"|")
					} else {
						templength := 256 - 3 - len(key)
						fmt.Fprintf(attributeWrite, key+" : "+value[:templength]+"|")
					}

				}
				fmt.Fprintf(attributeWrite, ",")

				for key, value := range vt.labelMap {
					if len(key) >= 250 {
						continue
					}
					value = strings.Replace(value, ",", " ", -1)
					if len(value)+3+len(key) < 256 {
						fmt.Fprintf(attributeWrite, key+" : "+value+"|")
					} else {
						templength := 256 - 3 - len(key)
						fmt.Fprintf(attributeWrite, key+" : "+value[:templength]+"|")
					}
				}

				if vc.cpuLimit == -1 {
					fmt.Fprintf(attributeWrite, ",")
				} else {
					fmt.Fprintf(attributeWrite, ",%d", vc.cpuLimit)
				}
				if vc.cpuRequest == -1 {
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
					fmt.Fprintf(attributeWrite, ",%s", time.Unix(vt.creationTime, 0).Format("2006-01-02 15:04:05.000"))
				}
				if vc.restarts == -1 {
					fmt.Fprintf(attributeWrite, ",,")
				} else {
					fmt.Fprintf(attributeWrite, ",%d,", vc.restarts)
				}
				for key, value := range vn.labelMap {
					if len(key) >= 250 {
						continue
					}
					value = strings.Replace(value, ",", " ", -1)
					if len(value)+3+len(key) < 256 {
						fmt.Fprintf(attributeWrite, key+" : "+value+"|")
					} else {
						templength := 256 - 3 - len(key)
						fmt.Fprintf(attributeWrite, key+" : "+value[:templength]+"|")
					}
				}
				// TODO: Not sure but order of these writes is different. Neet to check file format.
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
				if vn.memRequest == -1 {
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

//writeAttributes will create the attributes.csv file that is will be sent to Densify by the Forwarder.
func writeHPAAttributes(args *common.Parameters, systems map[string]map[string]string) {
	//Create the attributes file and open it for writing
	attributeWrite, err := os.Create("./data/hpa/hpa_extra_attributes.csv")
	if err != nil {
		args.ErrorLogger.Println("entity=" + entityKind + " message=" + err.Error())
		fmt.Println("[ERROR] entity=" + entityKind + " message=" + err.Error())
		return
	}

	//Write out the header.
	fmt.Fprintln(attributeWrite, "cluster,namespace,entity_name,entity_type,container,HPA Name,Labels")
	//Loop through the systems and write out the attributes data for each system.
	for i := range systems {
		//Write out the different fields. For fiels that are numeric we don't want to write -1 if it wasn't set so we write a blank if that is the value otherwise we write the number out.
		fmt.Fprintf(attributeWrite, "%s,%s,,,,%s,", *args.ClusterName, systems[i]["namespace"], i)
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
func writeWorkload(file io.Writer, result model.Value, namespace, pod, container model.LabelName, args *common.Parameters, kind string) {
	var tempKind bool
	if result == nil {
		return
	}
	if kind == "" {
		tempKind = true
	}

	//Loop through the results for the workload and validate that contains the required labels and that the entity exists in the systems data structure once validated will write out the workload for the system.
	for i := 0; i < result.(model.Matrix).Len(); i++ {
		if tempKind {
			kind = string(result.(model.Matrix)[i].Metric["owner_kind"])
		}
		namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]
		if !ok {
			continue
		}
		if _, ok := systems[string(namespaceValue)]; !ok {
			continue
		}
		podValue, ok := result.(model.Matrix)[i].Metric[pod]
		if !ok {
			continue
		}
		if _, ok := systems[string(namespaceValue)].midLevels[kind+"__"+string(podValue)]; !ok {
			continue
		}
		containerValue, ok := result.(model.Matrix)[i].Metric[container]
		if !ok {
			continue
		}
		if _, ok := systems[string(namespaceValue)].midLevels[kind+"__"+string(podValue)].containers[string(containerValue)]; !ok {
			continue
		}
		//Loop through the different values over the interval and write out each one to the workload file.
		for j := 0; j < len(result.(model.Matrix)[i].Values); j++ {
			fmt.Fprintf(file, "%s,%s,%s,%s,%s,%s,%f\n", *args.ClusterName, namespaceValue, systems[string(namespaceValue)].midLevels[kind+"__"+string(podValue)].name, systems[string(namespaceValue)].midLevels[kind+"__"+string(podValue)].kind, strings.Replace(string(containerValue), ":", ".", -1), time.Unix(0, int64(result.(model.Matrix)[i].Values[j].Timestamp)*1000000).Format("2006-01-02 15:04:05.000"), result.(model.Matrix)[i].Values[j].Value)
		}
	}
}

//writeWorkload will write out the workload data specific to metric provided to the file that was passed in.
func writeWorkloadMid(file io.Writer, result model.Value, namespace, mid model.LabelName, args *common.Parameters, prefix string) {
	if result == nil {
		return
	}

	//Loop through the results for the workload and validate that contains the required labels and that the entity exists in the systems data structure once validated will write out the workload for the system.
	for i := 0; i < result.(model.Matrix).Len(); i++ {
		namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]
		if !ok {
			continue
		}
		if _, ok := systems[string(namespaceValue)]; !ok {
			continue
		}
		midValue, ok := result.(model.Matrix)[i].Metric[mid]
		if !ok {
			continue
		}
		if _, ok := systems[string(namespaceValue)].pointers[prefix+"__"+string(midValue)]; !ok { //NOT PASSING THIS STATMENT
			continue
		}
		for kc := range systems[string(namespaceValue)].pointers[prefix+"__"+string(midValue)].containers {
			//Loop through the different values over the interval and write out each one to the workload file.
			for j := 0; j < len(result.(model.Matrix)[i].Values); j++ {
				fmt.Fprintf(file, "%s,%s,%s,%s,%s,%s,%f\n", *args.ClusterName, namespaceValue, systems[string(namespaceValue)].pointers[prefix+"__"+string(midValue)].name, systems[string(namespaceValue)].pointers[prefix+"__"+string(midValue)].kind, strings.Replace(string(kc), ":", ".", -1), time.Unix(0, int64(result.(model.Matrix)[i].Values[j].Timestamp)*1000000).Format("2006-01-02 15:04:05.000"), result.(model.Matrix)[i].Values[j].Value)
			}

		}
	}
}
