package datamodel

type ContainerCluster struct {
	Name       string                `json:"name,omitempty"`
	Namespaces map[string]*Namespace `json:"namespaces,omitempty"`
}

type Namespace struct {
	LabelMap LabelMap                        `json:"labels,omitempty"`
	Entities map[string]map[string]*MidLevel `json:"entities,omitempty"`
}

// MidLevel is used to hold information related to the highest owner of any containers
type MidLevel struct {
	OwnerName          *Labels               `json:"ownerName,omitempty"`
	OwnerKind          *Labels               `json:"ownerKind,omitempty"`
	NextSchedTime      *Labels               `json:"nextScheduledTime,omitempty"`
	StatusActive       *Labels               `json:"statusActive,omitempty"`
	LastSchedTime      *Labels               `json:"lastScheduledTime,omitempty"`
	MetadataGeneration *Labels               `json:"metadataGeneration,omitempty"`
	MaxSurge           *Labels               `json:"maxSurge,omitempty"`
	MaxUnavailable     *Labels               `json:"maxUnavailable,omitempty"`
	Completions        *Labels               `json:"completions,omitempty"`
	Parallelism        *Labels               `json:"parallelism,omitempty"`
	CompletionTime     *Labels               `json:"CompletionTime,omitempty"`
	CreationTime       *Labels               `json:"creationTime,omitempty"`
	Containers         map[string]*Container `json:"containers,omitempty"`
	LabelMap           LabelMap              `json:"labels,omitempty"`
}

const (
	SingleValueKey = "value"
)

// Container is used to hold information related to containers
type Container struct {
	CpuLimit   *Labels  `json:"cpuLimit,omitempty"`
	CpuRequest *Labels  `json:"cpuRequest,omitempty"`
	MemLimit   *Labels  `json:"memoryLimit,omitempty"`
	MemRequest *Labels  `json:"memoryRequest,omitempty"`
	PowerState *Labels  `json:"powerState,omitempty"`
	LabelMap   LabelMap `json:"labels,omitempty"`
}

type NodeCluster struct {
	Name  string           `json:"name,omitempty"`
	Nodes map[string]*Node `json:"nodes,omitempty"`
}

// Node is used  for storing attributes and config details
type Node struct {
	// Labels & general information about each node
	LabelMap         LabelMap `json:"labels,omitempty"`
	Roles            LabelMap `json:"roles,omitempty"`
	NetSpeedBytesMap LabelMap `json:"netSpeedBytesMap,omitempty"`
	AltWorkloadName  *Labels  `json:"altWorkloadName,omitempty"`
}

type RQCluster struct {
	Name       string                               `json:"name,omitempty"`
	Namespaces map[string]map[string]*ResourceQuota `json:"namespaces,omitempty"`
}

type ResourceQuota struct {
	CreationTime *Labels `json:"creationTime,omitempty"`
}

type CRQCluster struct {
	Name string                           `json:"name,omitempty"`
	CRQs map[string]*ClusterResourceQuota `json:"clusterResourceQuotas,omitempty"`
}

type ClusterResourceQuota struct {
	LabelMap      LabelMap `json:"labels,omitempty"`
	SelectorType  *Labels  `json:"selectorType,omitempty"`
	SelectorKey   *Labels  `json:"selectorKey,omitempty"`
	SelectorValue *Labels  `json:"selectorValue,omitempty"`
	CreationTime  *Labels  `json:"creationTime,omitempty"`
}
