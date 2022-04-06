package datamodel

import "time"

const (
	NamespaceMetricKey   = "namespace"
	NameKey              = "name"
	Unmapped             = "unmapped"
	PodIpKey             = "pod_ip"
	CompoundKeyDelimiter = "#@#"
)

type Discovery struct {
	ClusterName       string         `json:"clusterName,omitempty"`
	Range             *Range         `json:"range,omitempty"`
	MaxScrapeInterval *time.Duration `json:"maxScrapeInterval,omitempty"`
}

type Namespaces map[string]*Namespace

type ContainerDiscovery struct {
	Discovery  *Discovery `json:"discovery,omitempty"`
	Namespaces Namespaces `json:"namespaces,omitempty"`
}

type Entity struct {
	OwnerName    *Labels  `json:"ownerName,omitempty"`
	OwnerKind    *Labels  `json:"ownerKind,omitempty"`
	CreationTime *Labels  `json:"creationTime,omitempty"`
	DeletionTime *Labels  `json:"deletionTime,omitempty"`
	Generation   *Labels  `json:"generation,omitempty"`
	LabelMap     LabelMap `json:"labels,omitempty"`
}

type Entities map[string]*Entity

type EntityInterface interface {
	Get() *Entity
}

func (e *Entity) Get() *Entity {
	return e
}

type Namespace struct {
	*Entity
	CRDs                     map[string]Entities                 `json:"crds,omitempty"`                     // CRDs of unknown kinds: first key is kind, second is name
	HorizontalPodAutoscalers map[string]*HorizontalPodAutoscaler `json:"horizontalPodAutoscalers,omitempty"` // key is name
	CronJobs                 map[string]*CronJob                 `json:"cronjobs,omitempty"`                 // key is name
	Jobs                     map[string]*Job                     `json:"jobs,omitempty"`                     // key is name
	Deployments              map[string]*Deployment              `json:"deployments,omitempty"`              // key is name
	ReplicaSets              map[string]*ReplicaSet              `json:"replicaSets,omitempty"`              // key is name
	ReplicationControllers   map[string]*ReplicationController   `json:"replicationControllers,omitempty"`   // key is name
	StatefulSets             map[string]*StatefulSet             `json:"statefulSets,omitempty"`             // key is name
	DaemonSets               map[string]*DaemonSet               `json:"daemonSets,omitempty"`               // key is name
	Pods                     map[string]*Pod                     `json:"pods,omitempty"`                     // key is name
}

type ConditionManagement struct {
	Conditions    LabelMap              `json:"conditions,omitempty"`
	RawConditions map[string]*Condition `json:"-"`
}

type Pod struct {
	*Entity
	*ConditionManagement
	Containers map[string]*Container `json:"containers,omitempty"`
}

type ReplicationManagement struct {
	DesiredReplicas *Labels `json:"desiredReplicas,omitempty"`
	CurrentReplicas *Labels `json:"currentReplicas,omitempty"`
}

type ReplicaSet struct {
	*Entity
	*ReplicationManagement
}

type ReplicationController struct {
	*Entity
	*ReplicationManagement
}

type GenerationManagement struct {
	CurrentGeneration *Labels `json:"currentGeneration,omitempty"`
}

type StandardRollingUpdateStrategy struct {
	MaxSurge       *Labels `json:"maxSurge,omitempty"`
	MaxUnavailable *Labels `json:"maxUnavailable,omitempty"`
}

type Deployment struct {
	*Entity
	*ReplicationManagement
	*GenerationManagement
	*StandardRollingUpdateStrategy
	*ConditionManagement
	Paused *Labels `json:"paused,omitempty"`
}

type DaemonSet struct {
	*Entity
	*ReplicationManagement
	*GenerationManagement
	*StandardRollingUpdateStrategy
}

type StatefulSet struct {
	*Entity
	*ReplicationManagement
	*GenerationManagement
	// StatefulSetRollingUpdateStrategy is not a "standard" one
	Partition *Labels `json:"partition,omitempty"`
}

type Job struct {
	*Entity
	*ConditionManagement
	Completions    *Labels `json:"completions,omitempty"`
	Parallelism    *Labels `json:"parallelism,omitempty"`
	StartTime      *Labels `json:"startTime,omitempty"`
	CompletionTime *Labels `json:"completionTime,omitempty"`
}

type CronJob struct {
	*Entity
	Schedule                   *Labels `json:"schedule,omitempty"`
	ConcurrencyPolicy          *Labels `json:"concurrencyPolicy,omitempty"`
	Suspend                    *Labels `json:"suspend,omitempty"`
	SuccessfulJobsHistoryLimit *Labels `json:"successfulJobsHistoryLimit,omitempty"`
	FailedJobsHistoryLimit     *Labels `json:"failedJobsHistoryLimit,omitempty"`
	Active                     *Labels `json:"active,omitempty"`
	LastScheduledTime          *Labels `json:"lastScheduledTime,omitempty"`
	NextScheduledTime          *Labels `json:"nextScheduledTime,omitempty"`
}

type HorizontalPodAutoscaler struct {
	*Entity
	*ReplicationManagement
	*ConditionManagement
	TargetObjectKind *Labels `json:"targetObjectKind,omitempty"`
	TargetObjectName *Labels `json:"targetObjectName,omitempty"`
	MetricName       *Labels `json:"metricName,omitempty"`
	MetricTargetType *Labels `json:"metricTargetType,omitempty"`
	MinReplicas      *Labels `json:"minReplicas,omitempty"`
	MaxReplicas      *Labels `json:"maxReplicas,omitempty"`
}

const (
	SingleValueKey = "value"
)

// Container is used to hold information related to containers
type Container struct {
	Type       *ContainerType `json:"type,omitempty"`
	CpuLimit   *Labels        `json:"cpuLimit,omitempty"`
	CpuRequest *Labels        `json:"cpuRequest,omitempty"`
	MemLimit   *Labels        `json:"memoryLimit,omitempty"`
	MemRequest *Labels        `json:"memoryRequest,omitempty"`
	PowerState *Labels        `json:"powerState,omitempty"`
	LabelMap   LabelMap       `json:"labels,omitempty"`
}

type NodeDiscovery struct {
	Discovery *Discovery       `json:"discovery,omitempty"`
	Nodes     map[string]*Node `json:"nodes,omitempty"`
}

// Node is used for storing attributes and config details
type Node struct {
	*Entity
	*ConditionManagement
	// Labels & general information about each node
	Roles            LabelMap `json:"roles,omitempty"`
	NetSpeedBytesMap LabelMap `json:"netSpeedBytesMap,omitempty"`
	Capacity         LabelMap `json:"capacity,omitempty"`
	Allocatable      LabelMap `json:"allocatable,omitempty"`
}

type ResourceQuotaDiscovery struct {
	Discovery  *Discovery                           `json:"discovery,omitempty"`
	Namespaces map[string]map[string]*ResourceQuota `json:"namespaces,omitempty"`
}

type ResourceQuota struct {
	*Entity
}

type ClusterResourceQuotaDiscovery struct {
	Discovery *Discovery                       `json:"discovery,omitempty"`
	CRQs      map[string]*ClusterResourceQuota `json:"clusterResourceQuotas,omitempty"`
}

type ClusterResourceQuota struct {
	*Entity
	SelectorType  *Labels `json:"selectorType,omitempty"`
	SelectorKey   *Labels `json:"selectorKey,omitempty"`
	SelectorValue *Labels `json:"selectorValue,omitempty"`
}
