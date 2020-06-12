package common

import (
	"log"
	"time"
)

// Parameters - Reusable structure that holds common arguments used in the project
type Parameters struct {
	ClusterName, PromURL, PromAddress, FileName, Interval *string
	IntervalSize, History, Offset                         *int
	Debug                                                 bool
	CurrentTime                                           *time.Time
	LabelSuffix                                           string
	InfoLogger, WarnLogger, ErrorLogger, DebugLogger      *log.Logger
}
