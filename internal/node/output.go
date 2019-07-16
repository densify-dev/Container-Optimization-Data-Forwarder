/*
Used for outputting data and labels to CSVs. Mostly the same as container output but less checks
*/

//Package node output
package node

import (
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
	"github.com/prometheus/common/model"
)

//writeWorkload will write out the workload data specific to metric provided to the file that was passed in.
func writeWorkload(file io.Writer, result model.Value, node model.LabelName, promAddr, cluster string) {
	if result != nil {
		//Loop through the results for the workload and validate that contains the required labels and that the entity exists in the systems data structure once validated will write out the workload for the system.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			if nodeValue, ok := result.(model.Matrix)[i].Metric[node]; ok {
				if _, ok := nodes[string(nodeValue)]; ok {
					//Loop through the different values over the interval and write out each one to the workload file.
					for j := 0; j < len(result.(model.Matrix)[i].Values); j++ {
						var val model.SampleValue
						if math.IsNaN(float64(result.(model.Matrix)[i].Values[j].Value)) || math.IsInf(float64(result.(model.Matrix)[i].Values[j].Value), 0) {
							val = 0
						} else {
							val = result.(model.Matrix)[i].Values[j].Value
						}
						fmt.Fprintf(file, "%s,%s,%s,%f\n",
							cluster, strings.Replace(string(nodeValue), ";", ".", -1),
							time.Unix(0, int64(result.(model.Matrix)[i].Values[j].Timestamp)*1000000).Format("2006-01-02 15:04:05.000"),
							val)
					}
				}
			}
		}
	}
}

//writeConfig will create the config.csv file that is will be sent Densify by the Forwarder.
func writeConfig(clusterName, promAddr string) {
	var cluster string
	if clusterName == "" {
		cluster = promAddr
	} else {
		cluster = clusterName
	}

	//Create the config file and open it for writing.
	configWrite, err := os.Create("./data/node/config.csv")
	if err != nil {
		log.Println(prometheus.LogMessage("[ERROR]", promAddr, entityKind, "N/A", err.Error(), "N/A"))
	}

	//Write out the header.
	fmt.Fprintln(configWrite, "cluster,node,OS Name,HW Total CPUs,HW Total Physical CPUs,HW Cores Per CPU,HW Threads Per Core,HW Total Memory,BM Max Network IO Bps")

	//Loop through the nodes and write out the config data for each system.
	for kn := range nodes {
		fmt.Fprintf(configWrite, "%s,%s,%s", cluster, kn, nodes[kn].labelBetaKubernetesIoOs)

		if nodes[kn].cpuCapacity == -1 {
			fmt.Fprintf(configWrite, ",,")
		} else {
			fmt.Fprintf(configWrite, ",%d,%d", nodes[kn].cpuCapacity, nodes[kn].cpuCapacity)
		}

		fmt.Fprintf(configWrite, ",1,1")

		if nodes[kn].memCapacity == -1 {
			fmt.Fprintf(configWrite, ",")
		} else {
			fmt.Fprintf(configWrite, ",%d", nodes[kn].memCapacity)
		}

		if nodes[kn].netSpeedBytes == -1 {
			fmt.Fprintf(configWrite, ",")
		} else {
			fmt.Fprintf(configWrite, ",%d", nodes[kn].netSpeedBytes)
		}

		fmt.Fprintf(configWrite, "\n")
	}

}

//writeAttributes will create the attributes.csv file that is will be sent Densify by the Forwarder.
func writeAttributes(clusterName, promAddr string) {
	var cluster string
	if clusterName == "" {
		cluster = promAddr
	} else {
		cluster = clusterName
	}

	//Create the attributes file and open it for writing
	attributeWrite, err := os.Create("./data/node/attributes.csv")
	if err != nil {
		log.Println(prometheus.LogMessage("[ERROR]", promAddr, entityKind, "N/A", err.Error(), "N/A"))
	}

	//Write out the header.
	fmt.Fprintln(attributeWrite, "cluster,node,Virtual Technology,Virtual Domain,OS Architecture,Network Speed,Existing CPU Limit,Existing CPU Request,Existing Memory Limit,Existing Memory Request,Capacity Pods,Capacity CPU,Capacity Memory,Capacity Ephemeral Storage,Capacity Huge Pages,Allocatable Pods,Allocatable CPU,Allocatable Memory,Allocatable Ephemeral Storage,Allocatable Huge Pages,Node Labels")

	//Loop through the nodes and write out the attributes data for each system.
	for kn := range nodes {

		//Write out the different fields. For fiels that are numeric we don't want to write -1 if it wasn't set so we write a blank if that is the value otherwise we write the number out.
		fmt.Fprintf(attributeWrite, "%s,%s,Nodes,%s,%s", cluster, kn, cluster, nodes[kn].labelBetaKubernetesIoArch)

		if nodes[kn].netSpeedBytes == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, ",%d", nodes[kn].netSpeedBytes)
		}

		if nodes[kn].cpuLimit == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, ",%d", nodes[kn].cpuLimit)
		}

		if nodes[kn].cpuRequest == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, ",%d", nodes[kn].cpuRequest)
		}

		if nodes[kn].memLimit == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, ",%d", nodes[kn].memLimit)
		}

		if nodes[kn].memRequest == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, ",%d", nodes[kn].memRequest)
		}

		if nodes[kn].podsCapacity == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, ",%d", nodes[kn].podsCapacity)
		}

		if nodes[kn].cpuCapacity == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, ",%d", nodes[kn].cpuCapacity)
		}

		if nodes[kn].memCapacity == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, ",%d", nodes[kn].memCapacity)
		}

		if nodes[kn].ephemeralStorageCapacity == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, ",%d", nodes[kn].ephemeralStorageCapacity)
		}

		if nodes[kn].hugepages2MiCapacity == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, ",%d", nodes[kn].hugepages2MiCapacity)
		}

		if nodes[kn].podsAllocatable == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, ",%d", nodes[kn].podsAllocatable)
		}

		if nodes[kn].cpuAllocatable == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, ",%d", nodes[kn].cpuAllocatable)
		}

		if nodes[kn].memAllocatable == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, ",%d", nodes[kn].memAllocatable)
		}

		if nodes[kn].ephemeralStorageAllocatable == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, ",%d", nodes[kn].ephemeralStorageAllocatable)
		}

		if nodes[kn].hugepages2MiAllocatable == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, ",%d", nodes[kn].hugepages2MiAllocatable)
		}

		if nodes[kn].nodeLabel == "" {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, ",%s", nodes[kn].nodeLabel)
		}
		fmt.Fprintf(attributeWrite, "\n")

	}

}
