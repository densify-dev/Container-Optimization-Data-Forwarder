//Package cronjob collects data related to cronjobs and formats into csv files to send to Densify.
package cronjob

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/prometheus/common/model"
)

//writeConfigJob will create the config.csv file that is will be sent Densify by the Forwarder.
func writeConfigJob(clusterName, promAddr string) {
	//Create the config file and open it for writing.
	configWrite, err := os.Create("./data/job/config.csv")
	if err != nil {
		log.Println(err)
	}

	//Write out the header.
	fmt.Fprintln(configWrite, "cluster,namespace,job,cronjob,HW Total Memory,OS Name,HW Manufacturer,HW Model,HW Serial Number")
	//Check if the cluster parameter is set and if it is then use it for the name of the cluster if not use the prometheus address as the cluster name.
	var cluster string
	if clusterName == "" {
		cluster = promAddr
	} else {
		cluster = clusterName
	}
	//Loop through the systems and write out the config data for each system.
	for kn := range namespaces {
		for kc := range namespaces[kn].cronjobs {
			for kj := range namespaces[kn].cronjobs[kc].jobs {
				fmt.Fprintf(configWrite, "%s,%s,%s,%s,,Linux,CONTAINERS,%s,%s\n", cluster, kn, strings.Replace(kj, ";", ".", -1), strings.Replace(kc, ";", ".", -1), kn, kn)

			}
		}
	}
}

//writeAttributesJob will create the attributes.csv file that is will be sent Densify by the Forwarder.
func writeAttributesJob(clusterName, promAddr string) {
	//Create the attributes file and open it for writing
	attributeWrite, err := os.Create("./data/job/attributes.csv")
	if err != nil {
		log.Println(err)
	}

	//Write out the header.
	//fmt.Fprintln(attributeWrite, "cluster,namespace,pod,container,Virtual Technology,Virtual Domain,Virtual Datacenter,Virtual Cluster,Container Labels,Container Info,Pod Info,Pod Labels,Existing CPU Limit,Existing CPU Request,Existing Memory Limit,Existing Memory Request,Container Name,Current Nodes,Power State,Created By Kind,Created By Name,Current Size,Create Time,Container Restarts,Namespace Labels,Namespace CPU Request,Namespace CPU Limit,Namespace Memory Request,Namespace Memory Limit,Controller Labels")
	fmt.Fprintln(attributeWrite, "cluster,namespace,job,cronjob,Job Labels,Job Info,Virtual Technology,Virtual Domain,Virtual Datacenter,Virtual Cluster,Job Spec Completions,Job Spec Parallelism,Job Status Completion Time,Job Status Start Time")

	//Check if the cluster parameter is set and if it is then use it for the name of the cluster if not use the prometheus address as the cluster name.
	var cluster string
	if clusterName == "" {
		cluster = promAddr
	} else {
		cluster = clusterName
	}
	//Loop through the systems and write out the attributes data for each system.
	for kn := range namespaces {
		for kc := range namespaces[kn].cronjobs {
			for kj, vj := range namespaces[kn].cronjobs[kc].jobs {

				//Write out the different fields. For fiels that are numeric we don't want to write -1 if it wasn't set so we write a blank if that is the value otherwise we write the number out.
				fmt.Fprintf(attributeWrite, "%s,%s,%s,%s,%s,%s,CronJob,%s,%s,%s,%d,%d,%d,%d\n", cluster, kn, strings.Replace(kj, ";", ".", -1), strings.Replace(kc, ";", ".", -1), vj.jobLabel, vj.jobInfo, cluster, kn, kc, vj.specCompletions, vj.specParallelism, vj.statusCompletionTime, vj.statusStartTime)
			}
		}
	}
}

//writeConfigCronJob will create the config.csv file that is will be sent Densify by the Forwarder.
func writeConfigCronJob(clusterName, promAddr string) {
	//Create the config file and open it for writing.
	configWrite, err := os.Create("./data/cronjob/config.csv")
	if err != nil {
		log.Println(err)
	}

	//Write out the header.
	fmt.Fprintln(configWrite, "cluster,namespace,cronjob,HW Total Memory,OS Name,HW Manufacturer,HW Model,HW Serial Number")
	//Check if the cluster parameter is set and if it is then use it for the name of the cluster if not use the prometheus address as the cluster name.
	var cluster string
	if clusterName == "" {
		cluster = promAddr
	} else {
		cluster = clusterName
	}
	//Loop through the systems and write out the config data for each system.
	for kn := range namespaces {
		for kc := range namespaces[kn].cronjobs {
			if kc != "<none>" {
				fmt.Fprintf(configWrite, "%s,%s,%s,,Linux,CONTAINERS,%s,%s\n", cluster, kn, strings.Replace(kc, ";", ".", -1), kn, kn)
			}
		}
	}
}

//writeAttributesCronJob will create the attributes.csv file that is will be sent Densify by the Forwarder.
func writeAttributesCronJob(clusterName, promAddr string) {
	//Create the attributes file and open it for writing
	attributeWrite, err := os.Create("./data/cronjob/attributes.csv")
	if err != nil {
		log.Println(err)
	}

	//Write out the header.
	//fmt.Fprintln(attributeWrite, "cluster,namespace,pod,container,Virtual Technology,Virtual Domain,Virtual Datacenter,Virtual Cluster,Container Labels,Container Info,Pod Info,Pod Labels,Existing CPU Limit,Existing CPU Request,Existing Memory Limit,Existing Memory Request,Container Name,Current Nodes,Power State,Created By Kind,Created By Name,Current Size,Create Time,Container Restarts,Namespace Labels,Namespace CPU Request,Namespace CPU Limit,Namespace Memory Request,Namespace Memory Limit,Controller Labels")
	fmt.Fprintln(attributeWrite, "cluster,namespace,cronjob,CronJob Labels,CronJob Info,Virtual Technology,Virtual Domain,Virtual Datacenter,Virtual Cluster,Cron Job Next Schedule Time,Cron Job Last Schedule Time,Cron Job Active")

	//Check if the cluster parameter is set and if it is then use it for the name of the cluster if not use the prometheus address as the cluster name.
	var cluster string
	if clusterName == "" {
		cluster = promAddr
	} else {
		cluster = clusterName
	}
	//Loop through the systems and write out the attributes data for each system.
	for kn := range namespaces {
		for kc, vc := range namespaces[kn].cronjobs {
			if kc != "<none>" {
				//Write out the different fields. For fiels that are numeric we don't want to write -1 if it wasn't set so we write a blank if that is the value otherwise we write the number out.
				fmt.Fprintf(attributeWrite, "%s,%s,%s,%s,%s,CronJob,%s,%s,%s,%d,%d,%d\n", cluster, kn, strings.Replace(kc, ";", ".", -1), vc.cronjobLabel, vc.cronjobInfo, cluster, kn, kc, vc.nextScheduleTime, vc.lastScheduleTime, vc.statusActive)
			}
		}
	}
}

//writeWorkload will write out the workload data specific to metric provided to the file that was passed in.
func writeWorkload(file io.Writer, result model.Value, namespace, container model.LabelName, clusterName, promAddr string) {
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
					if cronjobValue, ok := result.(model.Matrix)[i].Metric["cronjob"]; ok {
						if _, ok := namespaces[string(namespaceValue)].cronjobs[string(cronjobValue)]; ok {
							//Loop through the different values over the interval and write out each one to the workload file.
							for j := 0; j < len(result.(model.Matrix)[i].Values); j++ {
								fmt.Fprintf(file, "%s,%s,%s,%s,%f\n", cluster, namespaceValue, strings.Replace(string(cronjobValue), ";", ".", -1), time.Unix(0, int64(result.(model.Matrix)[i].Values[j].Timestamp)*1000000).Format("2006-01-02 15:04:05.000"), result.(model.Matrix)[i].Values[j].Value)
							}
						}
					}
				}
			}
		}
	}
}
