/*
Used for outputting data and labels to CSVs. Mostly the same as container output but less checks
*/

//Package node output
package node

import (
	"fmt"
	"os"
	"strings"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/common"
)

//writeConfig will create the config.csv file that is will be sent to Densify by the Forwarder.
func writeConfig(args *common.Parameters) {

	//Create the config file and open it for writing.
	configWrite, err := os.Create("./data/node/config.csv")
	if err != nil {
		args.ErrorLogger.Println("entity=" + entityKind + " message=" + err.Error())
		fmt.Println("[ERROR] entity=" + entityKind + " message=" + err.Error())
		return
	}

	//Write out the header.
	fmt.Fprintln(configWrite, "ClusterName,NodeName,HwModel,OsName,HwTotalCpus,HwTotalPhysicalCpus,HwCoresPerCpu,HwThreadsPerCore,HwTotalMemory,HwMaxNetworkIoBps")

	//Loop through the nodes and write out the config data for each system.
	for kn := range nodes {
		var os, instance string
		if _, ok := nodes[kn].labelMap["label_kubernetes_io_os"]; ok {
			os = "label_kubernetes_io_os"
		} else {
			os = "label_beta_kubernetes_io_os"
		}

		if value, ok := nodes[kn].labelMap["label_node_kubernetes_io_instance_type"]; ok {
			instance = value
		} else if value, ok := nodes[kn].labelMap["label_beta_kubernetes_io_instance_type"]; ok {
			instance = value
		} else {
			instance = ""
		}

		fmt.Fprintf(configWrite, "%s,%s,%s,%s", *args.ClusterName, kn, instance, nodes[kn].labelMap[os])

		if nodes[kn].cpuCapacity == -1 {
			fmt.Fprintf(configWrite, ",,")
		} else {
			fmt.Fprintf(configWrite, ",%d,%d", nodes[kn].cpuCapacity, nodes[kn].cpuCapacity)
		}

		fmt.Fprintf(configWrite, ",1,1")

		if nodes[kn].memCapacity == -1 {
			fmt.Fprintf(configWrite, ",")
		} else {
			fmt.Fprintf(configWrite, ",%d", nodes[kn].memCapacity/1024/1024)
		}

		if nodes[kn].netSpeedBytes == -1 {
			fmt.Fprintf(configWrite, ",")
		} else {
			fmt.Fprintf(configWrite, ",%d", nodes[kn].netSpeedBytes)
		}

		fmt.Fprintf(configWrite, "\n")
	}
}

//writeAttributes will create the attributes.csv file that is will be sent to Densify by the Forwarder.
func writeAttributes(args *common.Parameters) {

	//Create the attributes file and open it for writing
	attributeWrite, err := os.Create("./data/node/attributes.csv")
	if err != nil {
		args.ErrorLogger.Println("entity=" + entityKind + " message=" + err.Error())
		fmt.Println("[ERROR] entity=" + entityKind + " message=" + err.Error())
		return
	}

	//Write out the header.
	fmt.Fprintln(attributeWrite, "ClusterName,NodeName,VirtualTechnology,VirtualDomain,VirtualDatacenter,VirtualCluster,OsArchitecture,NetworkSpeed,CpuLimit,CpuRequest,MemoryLimit,MemoryRequest,CapacityPods,CapacityCpu,CapacityMemory,CapacityEphemeralStorage,CapacityHugePages,AllocatablePods,AllocatableCpu,AllocatableMemory,AllocatableEphemeralStorage,AllocatableHugePages,NodeLabels")

	//Loop through the nodes and write out the attributes data for each system.
	for kn := range nodes {

		var beta, region, zone string
		if _, ok := nodes[kn].labelMap["label_kubernetes_io_arch"]; ok {
			beta = ""
		} else {
			beta = "beta_"
		}

		if value, ok := nodes[kn].labelMap["label_topology_kubernetes_io_region"]; ok {
			region = value
		} else if value, ok := nodes[kn].labelMap["label_failure_domain_beta_kubernetes_io_region"]; ok {
			region = value
		} else {
			region = ""
		}

		if value, ok := nodes[kn].labelMap["label_topology_kubernetes_io_zone"]; ok {
			zone = value
		} else if value, ok := nodes[kn].labelMap["label_failure_domain_beta_kubernetes_io_zone"]; ok {
			zone = value
		} else {
			zone = ""
		}

		//Write out the different fields. For fiels that are numeric we don't want to write -1 if it wasn't set so we write a blank if that is the value otherwise we write the number out.
		fmt.Fprintf(attributeWrite, "%s,%s,Nodes,%s,%s,%s,%s", *args.ClusterName, kn, *args.ClusterName, region, zone, nodes[kn].labelMap["label_"+beta+"kubernetes_io_arch"])

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
			fmt.Fprintf(attributeWrite, ",,")
		} else {
			fmt.Fprintf(attributeWrite, ",%d,", nodes[kn].hugepages2MiAllocatable)
		}

		for key, value := range nodes[kn].labelMap {
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
		fmt.Fprintf(attributeWrite, "\n")

	}
}
