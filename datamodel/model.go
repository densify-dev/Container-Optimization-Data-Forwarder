package datamodel

import "time"

type ContainerCluster struct {
	Name       string                `json:"name,omitempty"`
	Namespaces map[string]*Namespace `json:"namespaces,omitempty"`
}

type Namespace struct {
	LabelMap map[string]map[string]string    `json:"labels,omitempty"`
	Entities map[string]map[string]*MidLevel `json:"entities,omitempty"`
}

// MidLevel is used to hold information related to the highest owner of any containers
type MidLevel struct {
	OwnerName          string                       `json:"ownerName,omitempty"`
	OwnerKind          string                       `json:"ownerKind,omitempty"`
	NextSchedTime      int64                        `json:"nextScheduledTime,omitempty"`
	StatusActive       int64                        `json:"statusActive,omitempty"`
	LastSchedTime      int64                        `json:"lastScheduledTime,omitempty"`
	MetadataGeneration int64                        `json:"metadataGeneration,omitempty"`
	MaxSurge           int64                        `json:"maxSurge,omitempty"`
	MaxUnavailable     int64                        `json:"maxUnavailable,omitempty"`
	Completions        int64                        `json:"completions,omitempty"`
	Parallelism        int64                        `json:"parallelism,omitempty"`
	CompletionTime     int64                        `json:"CompletionTime,omitempty"`
	Containers         map[string]*Container        `json:"containers,omitempty"`
	CreationTime       int64                        `json:"creationTime,omitempty"`
	LabelMap           map[string]map[string]string `json:"labels,omitempty"`
}

// Container is used to hold information related to containers
type Container struct {
	CpuLimit   int                          `json:"cpuLimit,omitempty"`
	CpuRequest int                          `json:"cpuRequest,omitempty"`
	MemLimit   int                          `json:"memoryLimit,omitempty"`
	MemRequest int                          `json:"memoryRequest,omitempty"`
	PowerState int                          `json:"powerState,omitempty"`
	LabelMap   map[string]map[string]string `json:"labels,omitempty"`
}

type CRQCluster struct {
	Name string          `json:"name,omitempty"`
	CRQs map[string]*CRQ `json:"clusterResourceQuotas,omitempty"`
}

type CRQ struct {
	//Labels & general information about each node
	LabelMap      map[string]map[string]string `json:"labels,omitempty"`
	SelectorType  string                       `json:"selectorType,omitempty"`
	SelectorKey   string                       `json:"selectorKey,omitempty"`
	SelectorValue string                       `json:"selectorValue,omitempty"`
	Namespaces    string                       `json:"namespaces,omitempty"`
	CreateTime    time.Time                    `json:"creationTime,omitempty"`
}

type NodeCluster struct {
	Name  string           `json:"name,omitempty"`
	Nodes map[string]*Node `json:"nodes,omitempty"`
}

// Node is used  for storing attributes and config details
type Node struct {
	//Labels & general information about each node
	LabelMap        map[string]map[string]string `json:"labels,omitempty"`
	NetSpeedBytes   int                          `json:"netSpeedBytes,omitempty"`
	AltWorkloadName string                       `json:"altWorkloadName,omitempty"`
}

type RQCluster struct {
	Name       string                          `json:"name,omitempty"`
	Namespaces map[string]map[string]time.Time `json:"namespaces,omitempty"`
}
