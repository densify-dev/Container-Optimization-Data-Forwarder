//Package datacollection collects data from Prometheus and formats the data into CSVs that will be sent to Densify through the Forwarder.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	//"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/container"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/container2"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/node"
	"github.com/spf13/viper"
)

//Global variables used for Storing system info, command line\config file parameters.
var clusterName, promAddr, promPort, promProtocol, interval, configFile, configPath string
var intervalSize, history int
var debug bool
var currentTime time.Time

//initParamters will look for settings defined on the command line or in config.properties file and update accordingly. Also defines the default values for these variables.
//Note if the value is defined both on the command line and in the config.properties the value in the config.properties will be used.
func initParameters() {
	//Get the settings passed in from the command line and update the variables as required.
	flag.StringVar(&clusterName, "clusterName", "", "Name of the cluster to show in Densify")
	flag.StringVar(&promProtocol, "protocol", "http", "Which protocol to use http|https")
	flag.StringVar(&promAddr, "address", "", "Name of the Prometheus Server")
	flag.StringVar(&promPort, "port", "9090", "Prometheus Port")
	flag.StringVar(&interval, "interval", "hours", "Interval to use for data collection. Can be days, hours or minutes")
	flag.IntVar(&intervalSize, "intervalSize", 1, "Interval size to be used for querying. eg. default of 1 with default interval of hours queries 1 last hour of info")
	flag.IntVar(&history, "history", 1, "Amount of time to go back for data collection works with the interval and intervalSize settings")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.StringVar(&configFile, "file", "config", "Name of the config file without extention. Default config")
	flag.StringVar(&configPath, "path", "./config", "Path to where the config file is stored")
	flag.Parse()

	//Set defaults for viper to use if setting not found in the config.properties file.
	viper.SetDefault("cluster_name", clusterName)
	viper.SetDefault("prometheus_protocol", promProtocol)
	viper.SetDefault("prometheus_address", promAddr)
	viper.SetDefault("prometheus_port", promPort)
	viper.SetDefault("interval", interval)
	viper.SetDefault("interval_size", intervalSize)
	viper.SetDefault("history", history)
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
	debug = viper.GetBool("debug")

}

//main function.
func main() {

	//Open the debug log file for writing.
	debugLog, err := os.OpenFile("./data/log.txt", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer debugLog.Close()
	//Set log to use the debug log for writing output.
	log.SetOutput(debugLog)

	//Version number used for tracking which version of the code the client is using if there is an issue with data collection.
	log.Println("Version 1.0.0")

	//Read in the command line and config file parameters and set the required variables.
	initParameters()

	//Get the current time in UTC and format it. The script uses this time for all the queries this way if you have a large environment we are collecting the data as a snapshot of a specific time and not potentially getting a misaligned set of data.
	var t time.Time
	t = time.Now().UTC()
	if interval == "days" {
		currentTime = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	} else if interval == "hours" {
		currentTime = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
	} else {
		currentTime = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, t.Location())
	}

	//container.Metrics(clusterName, promProtocol, promAddr, promPort, interval, intervalSize, history, debug, currentTime)
	//deployment.Metrics(clusterName, promProtocol, promAddr, promPort, interval, intervalSize, history, debug, currentTime)
	//hpa.Metrics(clusterName, promProtocol, promAddr, promPort, interval, intervalSize, history, debug, currentTime)
	//cronjob.Metrics(clusterName, promProtocol, promAddr, promPort, interval, intervalSize, history, debug, currentTime)
	node.Metrics(clusterName, promProtocol, promAddr, promPort, interval, intervalSize, history, false, currentTime)
	//new_container_test.Metrics(clusterName, promProtocol, promAddr, promPort, interval, intervalSize, history, true, currentTime)
	container2.Metrics(clusterName, promProtocol, promAddr, promPort, interval, intervalSize, history, debug, currentTime)
}
