package common

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/datamodel"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
)

const (
	// Entity Kinds
	ContainerEntityKind = "container"
	NodeEntityKind      = "node"
	RQEntityKind        = "rq"
	CRQEntityKind       = "crq"
)

const (
	NamespaceKey = "namespace"
)

var NamespaceFilter = []string{NamespaceKey}

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

func (args *Parameters) ToDiscovery(entityKind string) (*datamodel.Discovery, error) {
	var disc *datamodel.Discovery
	var err error
	if maxScrapeInterval, ok := entityMaxScrapeIntervals[entityKind]; ok {
		var d time.Duration
		if d, err = time.ParseDuration(args.History); err == nil {
			start := args.CurrentTime.Add(-d)
			r := &datamodel.Range{Start: &start, End: args.CurrentTime}
			disc = &datamodel.Discovery{
				ClusterName:       *args.ClusterName,
				Range:             r,
				MaxScrapeInterval: maxScrapeInterval,
			}
		}
	} else {
		err = fmt.Errorf("unknown entity kind %s", entityKind)
	}
	return disc, err
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
	query = fmt.Sprintf("%s[%s]", query, args.History)
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

// Prometheus exporters
const (
	ne   = "node-exporter"
	cad  = "cadvisor"
	ksm  = "kube-state-metrics"
	ossm = "openshift-state-metrics"
)

// observed scrape intervals
// k8s cluster 	ne 	cad 	ksm 	ossm
// ------------ --- ------- ------- ------
// bare-metal	15s 25-46s	30s		N/A
// EKS 			60s 60s		60s		N/A
// GKE 			5s  12-25s	5s		N/A
// AKS 			30s 17-40s  30s 	N/A
// CRC 			15s 30s 	60s 	120s
// ------------ --- ------- ------- ------
// max + margin 75s 75s 	75s 	150s

var maxScrapeIntervals = map[string]time.Duration{
	ne:   time.Second * 75,
	cad:  time.Second * 75,
	ksm:  time.Second * 75,
	ossm: time.Second * 150,
}

var entityExporterMap = map[string][]string{
	ContainerEntityKind: {ksm, cad},
	NodeEntityKind:      {ksm, ne},
	RQEntityKind:        {ksm},
	CRQEntityKind:       {ossm},
}

var entityKinds = []string{ContainerEntityKind, NodeEntityKind, RQEntityKind, CRQEntityKind}

var entityMaxScrapeIntervals = initMaxScrapeIntervalMap()

func initMaxScrapeIntervalMap() map[string]*time.Duration {
	m := make(map[string]*time.Duration, len(entityKinds))
	for _, entityKind := range entityKinds {
		var maxInterval time.Duration
		exporters := entityExporterMap[entityKind]
		for _, exporter := range exporters {
			d := maxScrapeIntervals[exporter]
			if d > maxInterval {
				maxInterval = d
			}
		}
		m[entityKind] = &maxInterval
	}
	return m
}
