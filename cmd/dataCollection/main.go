//Package main collects data from Prometheus and formats the data into CSVs that will be sent to Densify through the Forwarder.
package main

import (
	"flag"
	"os"
	"strconv"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/cluster"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/container2"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/logger"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/node"
	"github.com/spf13/viper"
)

//Global variables used for Storing system info, command line\config file parameters.
var clusterName, promPort, promAddr, promProtocol, interval, configFile, configPath string
var intervalSize, history, offset int
var debug bool
var currentTime time.Time

//Temps
var clusterNameTemp, promAddrTemp, promPortTemp, promProtocolTemp, intervalTemp string
var intervalSizeTemp, historyTemp, offsetTemp int
var debugTemp bool

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
	if tempEnvVar, ok := os.LookupEnv("PROMETHEUS_CLUSTER"); ok {
		clusterName = tempEnvVar
	}

	if tempEnvVar, ok := os.LookupEnv("PROMETHEUS_PROTOCOL"); ok {
		promProtocol = tempEnvVar
	}

	if tempEnvVar, ok := os.LookupEnv("PROMETHEUS_ADDRESS"); ok {
		promAddr = tempEnvVar
	}

	if tempEnvVar, ok := os.LookupEnv("PROMETHEUS_PORT"); ok {
		promPort = tempEnvVar
	}

	if tempEnvVar, ok := os.LookupEnv("PROMETHEUS_INTERVAL"); ok {
		interval = tempEnvVar
	}

	if tempEnvVar, ok := os.LookupEnv("PROMETHEUS_INTERVALSIZE"); ok {
		intervalSizeTemp, err := strconv.ParseInt(tempEnvVar, 10, 64)
		if err == nil {
			intervalSize = int(intervalSizeTemp)
		}
	}

	if tempEnvVar, ok := os.LookupEnv("PROMETHEUS_HISTORY"); ok {
		historyTemp, err := strconv.ParseInt(tempEnvVar, 10, 64)
		if err == nil {
			history = int(historyTemp)
		}
	}

	if tempEnvVar, ok := os.LookupEnv("PROMETHEUS_OFFSET"); ok {
		offsetTemp, err := strconv.ParseInt(tempEnvVar, 10, 64)
		if err == nil {
			offset = int(offsetTemp)
		}
	}

	if tempEnvVar, ok := os.LookupEnv("PROMETHEUS_DEBUG"); ok {
		debugTemp, err := strconv.ParseBool(tempEnvVar)
		if err == nil {
			debug = debugTemp
		}
	}

	if tempEnvVar, ok := os.LookupEnv("PROMETHEUS_CONFIGFILE"); ok {
		configFile = tempEnvVar
	}

	if tempEnvVar, ok := os.LookupEnv("PROMETHEUS_CONFIGPATH"); ok {
		configPath = tempEnvVar
	}

	//Get the settings passed in from the command line and update the variables as required.
	flag.StringVar(&clusterNameTemp, "clusterName", clusterName, "Name of the cluster to show in Densify")
	flag.StringVar(&promProtocolTemp, "protocol", promProtocol, "Which protocol to use http|https")
	flag.StringVar(&promAddrTemp, "address", promAddr, "Name of the Prometheus Server")
	flag.StringVar(&promPortTemp, "port", promPort, "Prometheus Port")
	flag.StringVar(&intervalTemp, "interval", interval, "Interval to use for data collection. Can be days, hours or minutes")
	flag.IntVar(&intervalSizeTemp, "intervalSize", intervalSize, "Interval size to be used for querying. eg. default of 1 with default interval of hours queries 1 last hour of info")
	flag.IntVar(&historyTemp, "history", history, "Amount of time to go back for data collection works with the interval and intervalSize settings")
	flag.IntVar(&offsetTemp, "offset", offset, "Amount of units (based on interval value) to offset the data collection backwards in time")
	flag.BoolVar(&debugTemp, "debug", debug, "Enable debug logging")
	flag.StringVar(&configFile, "file", configFile, "Name of the config file without extention. Default config")
	flag.StringVar(&configPath, "path", configPath, "Path to where the config file is stored")
	flag.Parse()

	//Set defaults for viper to use if setting not found in the config.properties file.
	if configFile != "" {

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
		if err == nil {

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

		}
	}

	visitor := func(a *flag.Flag) {
		switch a.Name {
		case "clusterName":
			clusterName = clusterNameTemp
		case "protocol":
			promProtocol = promProtocolTemp
		case "address":
			promAddr = promAddrTemp
		case "port":
			promPort = promPortTemp
		case "interval":
			interval = intervalTemp
		case "intervalSize":
			intervalSize = intervalSizeTemp
		case "history":
			history = historyTemp
		case "offset":
			offset = offsetTemp
		case "debug":
			debug = debugTemp
		}
	}

	flag.Visit(visitor)

}

//main function.
func main() {
	errors := "Version 2.0.3"

	//Read in the command line and config file parameters and set the required variables.
	initParameters()
	logger.SetPromAddr(promAddr)

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

	errors += container2.Metrics(clusterName, promProtocol, promAddr, promPort, interval, intervalSize, history, debug, currentTime)
	errors += node.Metrics(clusterName, promProtocol, promAddr, promPort, interval, intervalSize, history, debug, currentTime)
	errors += cluster.Metrics(clusterName, promProtocol, promAddr, promPort, interval, intervalSize, history, debug, currentTime)

	//Open the debug log file for writing.
	debugLog, _ := os.OpenFile("./data/log.txt", os.O_WRONLY|os.O_CREATE, 0644)

	logger.PrintLog(errors, debugLog)
}
