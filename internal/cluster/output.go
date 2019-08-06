package cluster

import (
	"fmt"
	"io"
	"math"
	"os"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/logger"
	"github.com/prometheus/common/model"
)

//writeWorkload will write out the workload data specific to metric provided to the file that was passed in.
func writeWorkload(file io.Writer, result model.Value, promAddr, clusterN string) {
	if result != nil {
		//Loop through the different values over the interval and write out each one to the workload file.
		for j := 0; j < len(result.(model.Matrix)[0].Values); j++ {
			var val model.SampleValue
			if math.IsNaN(float64(result.(model.Matrix)[0].Values[j].Value)) || math.IsInf(float64(result.(model.Matrix)[0].Values[j].Value), 0) {
				val = 0
			} else {
				val = result.(model.Matrix)[0].Values[j].Value
			}
			fmt.Fprintf(file, "%s,%s,%f\n",
				clusterN,
				time.Unix(0, int64(result.(model.Matrix)[0].Values[j].Timestamp)*1000000).Format("2006-01-02 15:04:05.000"),
				val)
		}

	}
}

//writeConfig will create the config.csv file that is will be sent Densify by the Forwarder.
func writeConfig(clusterName, promAddr string) (logReturn string) {
	errors := ""
	var cluster string
	if clusterName == "" {
		cluster = promAddr
	} else {
		cluster = clusterName
	}

	//Create the config file and open it for writing.
	configWrite, err := os.Create("./data/cluster/config.csv")
	if err != nil {
		return logger.LogError(map[string]string{"entity": entityKind, "message": err.Error()}, "ERROR")
	}

	//Write out the header.
	fmt.Fprintln(configWrite, "cluster")

	fmt.Fprintf(configWrite, "%s", cluster)
	/**
	if clusterEntity.cpuLimit == -1 {
		fmt.Fprintf(configWrite, ",,")
	} else {
		fmt.Fprintf(configWrite, ",%d,", clusterEntity.cpuLimit)
	}*/

	fmt.Fprintf(configWrite, "\n")

	return errors

}

//writeAttributes will create the attributes.csv file that is will be sent Densify by the Forwarder.
func writeAttributes(clusterName, promAddr string) (logReturn string) {
	errors := ""
	var cluster string
	if clusterName == "" {
		cluster = promAddr
	} else {
		cluster = clusterName
	}

	//Create the attributes file and open it for writing
	attributeWrite, err := os.Create("./data/cluster/attributes.csv")
	if err != nil {
		return logger.LogError(map[string]string{"entity": entityKind, "message": err.Error()}, "ERROR")
	}

	//Write out the header.
	fmt.Fprintln(attributeWrite, "cluster,Virtual Technology,Virtual Domain,Existing CPU Limit,Existing CPU Request,Existing Memory Limit,Existing Memory Request")

	//Write out the different fields. For fiels that are numeric we don't want to write -1 if it wasn't set so we write a blank if that is the value otherwise we write the number out.
	fmt.Fprintf(attributeWrite, "%s,Clusters,%s", cluster, cluster)

	if clusterEntity.cpuLimit == -1 {
		fmt.Fprintf(attributeWrite, ",")
	} else {
		fmt.Fprintf(attributeWrite, ",%d", clusterEntity.cpuLimit)
	}

	if clusterEntity.cpuRequest == -1 {
		fmt.Fprintf(attributeWrite, ",")
	} else {
		fmt.Fprintf(attributeWrite, ",%d", clusterEntity.cpuRequest)
	}

	if clusterEntity.memLimit == -1 {
		fmt.Fprintf(attributeWrite, ",")
	} else {
		fmt.Fprintf(attributeWrite, ",%d", clusterEntity.memLimit)
	}

	if clusterEntity.memRequest == -1 {
		fmt.Fprintf(attributeWrite, ",")
	} else {
		fmt.Fprintf(attributeWrite, ",%d", clusterEntity.memRequest)
	}

	fmt.Fprintf(attributeWrite, "\n")

	return errors

}
