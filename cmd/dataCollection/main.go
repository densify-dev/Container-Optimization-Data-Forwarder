//Package main collects data from Prometheus and formats the data into CSVs that will be sent to Densify through the Forwarder.
package main

import (
	"flag"
	"fmt"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/datamodel"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/container"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/crq"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/node"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/resourcequota"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/spf13/viper"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	logFlag = log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile
)

// Global structure used to store Forwarder instance parameters
var params *prometheus.Parameters

// Parameters that allows user to control what levels they want to collect data on (quota, node, container)
var includeContainer, includeNode, includeQuota bool

//initParamters will look for settings defined on the command line or in config.properties file and update accordingly. Also defines the default values for these variables.
//Note if the value is defined both on the command line and in the config.properties the value in the config.properties will be used.
func initParameters(version string) {
	//Set default settings
	var clusterName string
	var promProtocol = "http"
	var promAddr string
	var promPort = "9090"
	var interval = "hours"
	var intervalSize = 1
	var offset int
	var debug = false
	var configFile = "config"
	var configPath = "./config"
	var include = "container,node,quota"
	var oAuthTokenPath = ""
	var caCertPath = ""

	//Temporary variables for processing flags
	var clusterNameTemp, promAddrTemp, promPortTemp, promProtocolTemp, intervalTemp, oAuthTokenPathTemp, caCertPathTemp, includeTemp string
	var intervalSizeTemp, offsetTemp int
	var debugTemp bool

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

	if tempEnvVar, ok := os.LookupEnv("PROMETHEUS_INCLUDE"); ok {
		include = tempEnvVar
	}

	if tempEnvVar, ok := os.LookupEnv("OAUTH_TOKEN"); ok {
		oAuthTokenPath = tempEnvVar
	}

	if tempEnvVar, ok := os.LookupEnv("CA_CERT"); ok {
		caCertPath = tempEnvVar
	}

	//Get the settings passed in from the command line and update the variables as required.
	flag.StringVar(&clusterNameTemp, "clusterName", clusterName, "Name of the cluster to show in Densify")
	flag.StringVar(&promProtocolTemp, "protocol", promProtocol, "Which protocol to use http|https")
	flag.StringVar(&promAddrTemp, "address", promAddr, "Name of the Prometheus Server")
	flag.StringVar(&promPortTemp, "port", promPort, "Prometheus Port")
	flag.StringVar(&intervalTemp, "interval", interval, "Interval to use for data collection. Can be days, hours or minutes")
	flag.IntVar(&intervalSizeTemp, "intervalSize", intervalSize, "Interval size to be used for querying. eg. default of 1 with default interval of hours queries 1 last hour of info")
	flag.IntVar(&offsetTemp, "offset", offset, "Amount of units (based on interval value) to offset the data collection backwards in time")
	flag.BoolVar(&debugTemp, "debug", debug, "Enable debug logging")
	flag.StringVar(&configFile, "file", configFile, "Name of the config file without extension. Default config")
	flag.StringVar(&configPath, "path", configPath, "Path to where the config file is stored")
	flag.StringVar(&includeTemp, "includeList", include, "Comma separated list of data to include in collection (node, container, quota) Ex: \"node,quota\"")
	flag.StringVar(&oAuthTokenPathTemp, "oAuthToken", oAuthTokenPath, "Path to oAuth token file required to authenticate with the Cluster where Prometheus is running.")
	flag.StringVar(&caCertPathTemp, "caCert", caCertPath, "Path to CA certificate required to pass certificate validation if using HTTPS")
	flag.Parse()

	//Set defaults for viper to use if setting not found in the config.properties file.
	if configFile != "" {

		viper.SetDefault("cluster_name", clusterName)
		viper.SetDefault("prometheus_protocol", promProtocol)
		viper.SetDefault("prometheus_address", promAddr)
		viper.SetDefault("prometheus_port", promPort)
		viper.SetDefault("interval", interval)
		viper.SetDefault("interval_size", intervalSize)
		viper.SetDefault("offset", offset)
		viper.SetDefault("debug", debug)
		viper.SetDefault("include_list", include)
		viper.SetDefault("prometheus_oauth_token", oAuthTokenPath)
		viper.SetDefault("ca_certificate", caCertPath)
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
			offset = viper.GetInt("offset")
			debug = viper.GetBool("debug")
			include = viper.GetString("include_list")
			oAuthTokenPath = viper.GetString("prometheus_oauth_token")
			caCertPath = viper.GetString("ca_certificate")
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
		case "offset":
			offset = offsetTemp
		case "debug":
			debug = debugTemp
		case "includeList":
			include = includeTemp
		case "oAuthToken":
			oAuthTokenPath = oAuthTokenPathTemp
		case "caCert":
			caCertPath = caCertPathTemp
		}
	}

	flag.Visit(visitor)

	promURL := promProtocol + "://" + promAddr + ":" + promPort

	logFile, err := os.OpenFile("./data/log.txt", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}

	var infoLogger, warnLogger, errorLogger, debugLogger *log.Logger

	infoLogger = log.New(logFile, "[INFO] ", logFlag)
	warnLogger = log.New(logFile, "[WARN] ", logFlag)
	errorLogger = log.New(logFile, "[ERROR] ", logFlag)
	debugLogger = log.New(logFile, "[DEBUG] ", logFlag)

	infoLogger.Printf("Version %s\n", version)
	fmt.Printf("[INFO] Version %s\n", version)

	// Check if token and certificate are missing
	if oAuthTokenPath != "" {
		if _, err := os.Stat(oAuthTokenPath); os.IsNotExist(err) {
			fmt.Printf("[INFO] %s does not exist. Attempting to execute without using oAuth token!\n", oAuthTokenPath)
			infoLogger.Printf("%s does not exist. Attempting to execute without using oAuth token!\n", oAuthTokenPath)
			oAuthTokenPath = ""
		}
	}

	if caCertPath != "" {
		if _, err := os.Stat(caCertPath); os.IsNotExist(err) {
			fmt.Printf("[INFO] %s does not exist. Attempting to execute without trusted CA Certificate configuration!\n", caCertPath)
			infoLogger.Printf("%s does not exist. Attempting to execute without trusted CA Certificate configuration!\n", caCertPath)
			caCertPath = ""
		}
	}

	if clusterName == "" {
		clusterName = promAddr
	}

	params = &prometheus.Parameters{

		ClusterName:    &clusterName,
		PromURL:        &promURL,
		Debug:          debug,
		InfoLogger:     infoLogger,
		WarnLogger:     warnLogger,
		ErrorLogger:    errorLogger,
		DebugLogger:    debugLogger,
		OAuthTokenPath: oAuthTokenPath,
		CaCertPath:     caCertPath,
	}
	parseIncludeParam(include)
	getVersion()
	timeRange(interval, intervalSize, offset)
}

func parseIncludeParam(param string) {
	param = strings.ToLower(param)
	for _, elem := range strings.Split(param, ",") {
		if strings.Compare(elem, "node") == 0 {
			includeNode = true
		} else if strings.Compare(elem, "container") == 0 {
			includeContainer = true
		} else if strings.Compare(elem, "quota") == 0 {
			includeQuota = true
		}
	}
}

func getVersion() {
	if ver, err := prometheus.GetVersion(params); err == nil {
		params.Version = ver
		fmt.Printf("[INFO] Detected Prometheus version %s\n", ver)
		params.InfoLogger.Printf("Detected Prometheus version %s\n", ver)
	}
}

const (
	// See: https://github.com/prometheus/prometheus/releases/tag/v2.21.0,
	// https://github.com/prometheus/prometheus/pull/7713
	minimumVersionCompositeDuration = "2.21.0"
	hour                            = "h"
	minute                          = "m"
	millisecond                     = "ms"
)

//timeRange allows you to define the start and end values of the range will pass to the Prometheus for the query.
func timeRange(interval string, intervalSize, offset int) {
	timeUTC := time.Now().UTC()
	y := timeUTC.Year()
	M := timeUTC.Month()
	d := timeUTC.Day()
	var h, m int
	// need to deal with non-composite and non-ms duration (pre minimumVersionCompositeDuration)
	ival := int64(intervalSize)
	var ivalUnit string
	if interval == "days" {
		d -= offset
		ival *= 24
		ivalUnit = hour
	} else if interval == "hours" {
		h = timeUTC.Hour() - offset
		ivalUnit = hour
	} else {
		// minutes
		h = timeUTC.Hour()
		m = timeUTC.Minute() - offset
		ivalUnit = minute
	}
	currentTime := time.Date(y, M, d, h, m, 0, 0, timeUTC.Location())
	params.CurrentTime = &currentTime
	params.History = fmt.Sprintf("%d%s", ival, ivalUnit)
	var dur time.Duration
	var err error
	var adjusted bool
	if dur, err = time.ParseDuration(params.History); err == nil {
		if params.Version >= minimumVersionCompositeDuration {
			dur -= datamodel.PrometheusGranularity
			ival = dur.Milliseconds()
			ivalUnit = millisecond
			params.History = fmt.Sprintf("%d%s", ival, ivalUnit)
			adjusted = true
		}
	}
	params.StartTimeInclusive = !adjusted
	start := currentTime.Add(-dur)
	params.StartTime = &start
	params.Range5Min = v1.Range{Start: start, End: currentTime, Step: time.Minute * 5}
}

//main function.
func main() {
	//Read in the command line and config file parameters and set the required variables.
	initParameters("3.0.0-beta")

	if includeContainer {
		container.Metrics(params)
	} else {
		params.InfoLogger.Println("Skipping container data collection")
		fmt.Println("[INFO] Skipping container data collection")
	}
	if includeNode {
		node.Metrics(params)
	} else {
		params.InfoLogger.Println("Skipping node data collection")
		fmt.Println("[INFO] Skipping node data collection")
	}
	if includeQuota {
		crq.Metrics(params)
		resourcequota.Metrics(params)
	} else {
		params.InfoLogger.Println("Skipping quota data collection")
		fmt.Println("[INFO] Skipping quota data collection")
	}
}
