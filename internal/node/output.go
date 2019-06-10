/*
Used for outputting data and labels to CSVs. Mostly the same as container output but less checks
*/

//Package node output
package node

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/prometheus/common/model"
)

//writeWorkload will write out the workload data specific to metric provided to the file that was passed in.
func writeWorkload(file io.Writer, result model.Value, node model.LabelName, promAddr string) {
	if result != nil {
		//Loop through the results for the workload and validate that contains the required labels and that the entity exists in the systems data structure once validated will write out the workload for the system.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			if nodeValue, ok := result.(model.Matrix)[i].Metric[node]; ok {
				if _, ok := nodes[string(nodeValue)]; ok {
					//Loop through the different values over the interval and write out each one to the workload file.
					for j := 0; j < len(result.(model.Matrix)[i].Values); j++ {
						fmt.Fprintf(file, "%s,%s,%s,%f\n",
							nodes[string(nodeValue)].namespace,
							strings.Replace(string(nodeValue), ";", ".", -1),
							time.Unix(0, int64(result.(model.Matrix)[i].Values[j].Timestamp)*1000000).Format("2006-01-02 15:04:05.000"),
							result.(model.Matrix)[i].Values[j].Value)
					}
				}
			}
		}
	}
}

//writeConfig will create the config.csv file that is will be sent Densify by the Forwarder.
func writeConfig(promAddr string) {
	//Create the config file and open it for writing.
	configWrite, err := os.Create("./data/node/config.csv")
	if err != nil {
		log.Println(err)
	}

	//Write out the header.
	fmt.Fprintln(configWrite, "namespace,node,label_beta_kubernetes_io_os,label_kubernetes_io_hostname")

	//Loop through the nodes and write out the config data for each system.
	for kn := range nodes {
		fmt.Fprintf(configWrite, "%s,%s,%s,%s\n", nodes[kn].namespace, kn, nodes[kn].labelBetaKubernetesIoOs, nodes[kn].labelKubernetesIoHostname)
	}
}

//writeAttributes will create the attributes.csv file that is will be sent Densify by the Forwarder.
func writeAttributes(promAddr string) {
	//Create the attributes file and open it for writing
	attributeWrite, err := os.Create("./data/node/attributes.csv")
	if err != nil {
		log.Println(err)
	}

	//Write out the header.
	fmt.Fprintln(attributeWrite, "namespace, node, label_beta_kubernetes_io_arch, network_speed_bytes, capacity, allocatable, labels")

	//Loop through the nodes and write out the attributes data for each system.
	for kn := range nodes {

		//Write out the different fields. For fiels that are numeric we don't want to write -1 if it wasn't set so we write a blank if that is the value otherwise we write the number out.
		fmt.Fprintf(attributeWrite, "%s,%s,%s,%d,%d,%d,%s\n", nodes[kn].namespace, kn, nodes[kn].labelBetaKubernetesIoArch, nodes[kn].netSpeedBytes, nodes[kn].capacity, nodes[kn].allocatable, nodes[kn].nodeLabel)
	}
}
