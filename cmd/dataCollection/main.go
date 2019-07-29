//Package datacollection collects data from Prometheus and formats the data into CSVs that will be sent to Densify through the Forwarder.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/cluster"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/container2"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/node"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
	"github.com/spf13/viper"
)

//Global variables used for Storing system info, command line\config file parameters.
var clusterName, promAddr, promPort, promProtocol, interval, configFile, configPath string
var intervalSize, history, offset int
var debug bool
var currentTime time.Time

//initParamters will look for settings defined on the command line or in config.properties file and update accordingly. Also defines the default values for these variables.
//Note if the value is defined both on the command line and in the config.properties the value in the config.properties will be used.
func initParameters() {
	//Set default settings
	clusterName = ""
	promProtocol = "http"
	promAddr = ""
	promPort = "9090"
	interval = "hours"
	intervalSize = 1
	history = 1
	offset = 0
	debug = false
	configFile = "config"
	configPath = "./config"

	//Set settings using environment variables
	exists := false
	tempEnvVar := ""

	tempEnvVar, exists = os.LookupEnv("PROMETHEUS_CLUSTER")
	if exists {
		clusterName = tempEnvVar
	}
	tempEnvVar, exists = os.LookupEnv("PROMETHEUS_PROTOCOL")
	if exists {
		promProtocol = tempEnvVar
	}
	tempEnvVar, exists = os.LookupEnv("PROMETHEUS_ADDRESS")
	if exists {
		promAddr = tempEnvVar
	}
	tempEnvVar, exists = os.LookupEnv("PROMETHEUS_PORT")
	if exists {
		promPort = tempEnvVar
	}
	tempEnvVar, exists = os.LookupEnv("PROMETHEUS_INTERVAL")
	if exists {
		interval = tempEnvVar
	}
	tempEnvVar, exists = os.LookupEnv("PROMETHEUS_INTERVALSIZE")
	if exists {
		intervalSizeTemp, err := strconv.ParseInt(tempEnvVar, 10, 64)
		if err == nil {
			intervalSize = int(intervalSizeTemp)
		}
	}
	tempEnvVar, exists = os.LookupEnv("PROMETHEUS_HISTORY")
	if exists {
		historyTemp, err := strconv.ParseInt(tempEnvVar, 10, 64)
		if err == nil {
			history = int(historyTemp)
		}
	}
	tempEnvVar, exists = os.LookupEnv("PROMETHEUS_OFFSET")
	if exists {
		offsetTemp, err := strconv.ParseInt(tempEnvVar, 10, 64)
		if err == nil {
			offset = int(offsetTemp)
		}
	}
	tempEnvVar, exists = os.LookupEnv("PROMETHEUS_DEBUG")
	if exists {
		debugTemp, err := strconv.ParseBool(tempEnvVar)
		if err == nil {
			debug = debugTemp
		}
	}
	tempEnvVar, exists = os.LookupEnv("PROMETHEUS_CONFIGFILE")
	if exists {
		configFile = tempEnvVar
	}
	tempEnvVar, exists = os.LookupEnv("PROMETHEUS_CONFIGPATH")
	if exists {
		configPath = tempEnvVar
	}

	//Set defaults for viper to use if setting not found in the config.properties file.
	viper.SetDefault("cluster_name", clusterName)
	viper.SetDefault("prometheus_protocol", promProtocol)
	viper.SetDefault("prometheus_address", promAddr)
	viper.SetDefault("prometheus_port", promPort)
	viper.SetDefault("interval", interval)
	viper.SetDefault("interval_size", intervalSize)
	viper.SetDefault("history", history)
	viper.SetDefault("offset", offset)
	viper.SetDefault("debug", debug)
	// Config import setup.
	viper.SetConfigName(configFile)
	viper.AddConfigPath(configPath)
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s ", err))
	}

	//Process the config.properties file update the variables as required.
	clusterName = viper.GetString("cluster_name")
	promProtocol = viper.GetString("prometheus_protocol")
	promAddr = viper.GetString("prometheus_address")
	promPort = viper.GetString("prometheus_port")
	interval = viper.GetString("interval")
	intervalSize = viper.GetInt("interval_size")
	history = viper.GetInt("history")
	offset = viper.GetInt("offset")
	debug = viper.GetBool("debug")

	//Get the settings passed in from the command line and update the variables as required.
	flag.StringVar(&clusterName, "clusterName", clusterName, "Name of the cluster to show in Densify")
	flag.StringVar(&promProtocol, "protocol", promProtocol, "Which protocol to use http|https")
	flag.StringVar(&promAddr, "address", promAddr, "Name of the Prometheus Server")
	flag.StringVar(&promPort, "port", promPort, "Prometheus Port")
	flag.StringVar(&interval, "interval", interval, "Interval to use for data collection. Can be days, hours or minutes")
	flag.IntVar(&intervalSize, "intervalSize", intervalSize, "Interval size to be used for querying. eg. default of 1 with default interval of hours queries 1 last hour of info")
	flag.IntVar(&history, "history", history, "Amount of time to go back for data collection works with the interval and intervalSize settings")
	flag.IntVar(&offset, "offset", offset, "Amount of units (based on interval value) to offset the data collection backwards in time")
	flag.BoolVar(&debug, "debug", debug, "Enable debug logging")
	flag.StringVar(&configFile, "file", configFile, "Name of the config file without extention. Default config")
	flag.StringVar(&configPath, "path", configPath, "Path to where the config file is stored")
	flag.Parse()

}

//main function.
func main() {

	//Open the debug log file for writing.
	debugLog, err := os.OpenFile("./data/log.txt", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(prometheus.LogMessage("[ERROR]", promAddr, "Main", "N/A", err.Error(), "N/A"))
	}
	defer debugLog.Close()
	//Set log to use the debug log for writing output.
	log.SetOutput(debugLog)
	log.SetFlags(0)
	log.SetPrefix(time.Now().Format(time.RFC3339Nano + " "))

	//Version number used for tracking which version of the code the client is using if there is an issue with data collection.
	log.Println("Version 2.0.1-beta")

	//Read in the command line and config file parameters and set the required variables.
	initParameters()

	//Get the current time in UTC and format it. The script uses this time for all the queries this way if you have a large environment we are collecting the data as a snapshot of a specific time and not potentially getting a misaligned set of data.
	var t time.Time
	t = time.Now().UTC()

	if interval == "days" {
		currentTime = time.Date(t.Year(), t.Month(), t.Day()-offset, 0, 0, 0, 0, t.Location())
	} else if interval == "hours" {
		currentTime = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()-offset, 0, 0, 0, t.Location())
	} else {
		currentTime = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute()-offset, 0, 0, t.Location())
	}

	container2.Metrics(clusterName, promProtocol, promAddr, promPort, interval, intervalSize, history, debug, currentTime)
	node.Metrics(clusterName, promProtocol, promAddr, promPort, interval, intervalSize, history, debug, currentTime)
	cluster.Metrics(clusterName, promProtocol, promAddr, promPort, interval, intervalSize, history, debug, currentTime)
}
