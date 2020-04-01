package common

// ARGS - Reusable structure that holds common arguments used in the project
type ARGS struct {
	ClusterName, PromAddress, FileName, MetricName, Query, Interval, Prefix, Metric *string
	IntervalSize, History                                                           *int
}
