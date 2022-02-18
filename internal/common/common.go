package common

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
)

// Parameters - Reusable structure that holds common arguments used in the project
type Parameters struct {
	ClusterName, PromURL                             *string
	Debug                                            bool
	CurrentTime                                      *time.Time
	InfoLogger, WarnLogger, ErrorLogger, DebugLogger *log.Logger
	OAuthTokenPath                                   string
	CaCertPath                                       string
	Range5Min                                        v1.Range
	History                                          string
}

//MetricCollect is used to query Prometheus to get data for specific query and return the results to be processed.
func MetricCollect(args *Parameters, query string) (value model.Value, err error) {
	//setup the context to use for the API calls
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tlsClientConfig := &tls.Config{}
	if args.CaCertPath != "" {
		tmpTLSConfig, err := config.NewTLSConfig(&config.TLSConfig{
			CAFile: args.CaCertPath,
		})
		if err != nil {
			log.Fatalf("Failed to generate TLS config:%v", err)
		}
		tlsClientConfig = tmpTLSConfig
	}

	var roundTripper http.RoundTripper = &http.Transport{
		TLSClientConfig: tlsClientConfig,
	}

	if args.OAuthTokenPath != "" {
		roundTripper = config.NewAuthorizationCredentialsFileRoundTripper("Bearer", args.OAuthTokenPath, roundTripper)
	}
	//Setup the API client connection
	client, err := api.NewClient(api.Config{Address: *args.PromURL, RoundTripper: roundTripper})
	if err != nil {
		return value, err
	}

	//Query prometheus with the values defined above as well as the query that was passed into the function.
	q := v1.NewAPI(client)
	// use Query API, not QueryRange for both discovery and workload
	query += `[` + args.History + `]`
	value, _, err = q.Query(ctx, query, *args.CurrentTime)
	if err != nil {
		return value, err
	}

	//If the values from the query return no data (length of 0) then give a warning
	if value == nil {
		err = errors.New("No resultset returned")
	} else if value.(model.Matrix) == nil {
		err = errors.New("No time series data returned")
	} else if value.(model.Matrix).Len() == 0 {
		err = errors.New("No data returned, value.(model.Matrix) is empty")
	}

	//Return the data that was received from Prometheus.
	return value, err
}

/*
// AddToLabelMap used to add values to label map used for attributes.
func AddToLabelMap(key string, value string, labelPath map[string]string) {
	if _, ok := labelPath[key]; !ok {
		value = strings.Replace(value, "\n", "", -1)
		value = strings.Replace(value, "\r", "", -1)
		if len(value) > 255 {
			labelPath[key] = value[:255]
		} else {
			labelPath[key] = value
		}
		return
	}

	if strings.Contains(value, ";") {
		currValue := ""
		for _, l := range value {
			currValue = currValue + string(l)
			if l == ';' {
				AddToLabelMap(key, currValue[:len(currValue)-1], labelPath)
				currValue = ""
			}
		}
		AddToLabelMap(key, currValue, labelPath)
		return
	}

	currValue := ""
	notPresent := true
	for _, l := range labelPath[key] {
		currValue = currValue + string(l)
		if l == ';' {
			if currValue[:len(currValue)-1] == value {
				notPresent = false
				break
			}
			currValue = ""
		}
	}
	if currValue != value && notPresent {
		if len(value) > 255 {
			labelPath[key] = labelPath[key] + ";" + value[:255]
		} else {
			labelPath[key] = labelPath[key] + ";" + value
		}
	}
}
*/

//GetWorkload used to query for the workload data and then calls write workload
func GetWorkload(fileName, query string, args *Parameters, entityKind string) {
	result, err := MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=" + fileName + " query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=" + fileName + " query=" + query + " message=" + err.Error())
	} else {
		file, _ := json.Marshal(result)
		err = os.WriteFile("./data/"+entityKind+"/"+fileName+".json", file, 0644)
		if err != nil {
			args.ErrorLogger.Println("entity=" + entityKind + " message=" + err.Error())
			fmt.Println("[ERROR] entity=" + entityKind + " message=" + err.Error())
		}
	}
}

//WriteDiscovery will create the attributes.csv file that is will be sent to Densify by the Forwarder.
func WriteDiscovery(args *Parameters, entityInterface interface{}, entityKind string) {
	//Create the discovery file.
	file, _ := json.Marshal(entityInterface)
	err := os.WriteFile("./data/"+entityKind+"/discovery.json", file, 0644)
	if err != nil {
		args.ErrorLogger.Println("entity=" + entityKind + " message=" + err.Error())
		fmt.Println("[ERROR] entity=" + entityKind + " message=" + err.Error())
	}
}
