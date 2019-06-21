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
						if math.IsNaN(float64(result.(model.Matrix)[i].Values[j].Value)) {
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
		log.Println(err)
	}

	//Write out the header.
	fmt.Fprintln(configWrite, "cluster,node,OS Name,HW Total CPUs,HW Total Physical CPUs,HW Cores Per CPU,HW Threads Per Core,HW Total Memory,BM Max Network IO Bps")

	//Loop through the nodes and write out the config data for each system.
	for kn := range nodes {
		fmt.Fprintf(configWrite, "%s,%s,%s,%d,%d,1,1,%d\n", cluster, kn, nodes[kn].labelBetaKubernetesIoOs, nodes[kn].cpuCapacity, nodes[kn].cpuCapacity, nodes[kn].memCapacity, nodes[kn].netSpeedBytes)
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
		log.Println(err)
	}

	//Write out the header.
	fmt.Fprintln(attributeWrite, "cluster,node,OS Architecture,Network Speed,Capacity Pods,Capacity CPU,Capacity Memory,Capacity Ephemeral Storage,Capacity Huge Pages,Allocatable Pods,Allocatable CPU,Allocatable Memory,Allocatable Ephemeral Storage,Allocatable Huge Pages,Node Labels")

	//Loop through the nodes and write out the attributes data for each system.
	for kn := range nodes {

		//Write out the different fields. For fiels that are numeric we don't want to write -1 if it wasn't set so we write a blank if that is the value otherwise we write the number out.
		fmt.Fprintf(attributeWrite, "%s,%s,%s,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%s\n", cluster, kn, nodes[kn].labelBetaKubernetesIoArch, nodes[kn].netSpeedBytes,
			nodes[kn].podsCapacity, nodes[kn].cpuCapacity, nodes[kn].memCapacity, nodes[kn].ephemeralStorageCapacity, nodes[kn].hugepages2MiCapacity,
			nodes[kn].podsAllocatable, nodes[kn].cpuAllocatable, nodes[kn].memAllocatable, nodes[kn].ephemeralStorageAllocatable, nodes[kn].hugepages2MiAllocatable,
			nodes[kn].nodeLabel)
	}
}
