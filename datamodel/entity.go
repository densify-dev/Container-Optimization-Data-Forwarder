package datamodel

const (
	NamespaceKey   = "Namespace"
	PodKey         = "Pod"
	ReplicaSetKey  = "ReplicaSet"
	RCKey          = "ReplicationController"
	DeploymentKey  = "Deployment"
	DaemonSetKey   = "DaemonSet"
	StatefulSetKey = "StatefulSet"
	JobKey         = "Job"
	CronJobKey     = "CronJob"
	HPAKey         = "HPA"
)

func GetEntity(namespaces Namespaces, namespace, kind, name string) (ei EntityInterface, f bool) {
	return GetOrCreateEntity(namespaces, namespace, kind, name, false)
}

func EnsureEntity(namespaces Namespaces, namespace, kind, name string) (ei EntityInterface, f bool) {
	return GetOrCreateEntity(namespaces, namespace, kind, name, true)
}

func GetOrCreateEntity(namespaces Namespaces, namespace, kind, name string, force bool) (ei EntityInterface, f bool) {
	if ns, ok := namespaces[namespace]; ok {
		switch kind {
		case NamespaceKey:
			ei, f = ns, ok
		case PodKey:
			ei, f = ns.Pods[name]
			if force && !f {
				p := newPod()
				ns.Pods[name] = p
				ei = p
				f = true
			}
		case ReplicaSetKey:
			ei, f = ns.ReplicaSets[name]
			if force && !f {
				rs := newReplicaSet()
				ns.ReplicaSets[name] = rs
				ei = rs
				f = true
			}
		case RCKey:
			ei, f = ns.ReplicationControllers[name]
			if force && !f {
				rc := newReplicationController()
				ns.ReplicationControllers[name] = rc
				ei = rc
				f = true
			}
		case DeploymentKey:
			ei, f = ns.Deployments[name]
			if force && !f {
				d := newDeployment()
				ns.Deployments[name] = d
				ei = d
				f = true
			}
		case DaemonSetKey:
			ei, f = ns.DaemonSets[name]
			if force && !f {
				ds := newDaemonSet()
				ns.DaemonSets[name] = ds
				ei = ds
				f = true
			}
		case StatefulSetKey:
			ei, f = ns.StatefulSets[name]
			if force && !f {
				ss := newStatefulSet()
				ns.StatefulSets[name] = ss
				ei = ss
				f = true
			}
		case JobKey:
			ei, f = ns.Jobs[name]
			if force && !f {
				j := newJob()
				ns.Jobs[name] = j
				ei = j
				f = true
			}
		case CronJobKey:
			ei, f = ns.CronJobs[name]
			if force && !f {
				cj := newCronJob()
				ns.CronJobs[name] = cj
				ei = cj
				f = true
			}
		case HPAKey:
			ei, f = ns.HorizontalPodAutoscalers[name]
			if force && !f {
				hpa := newHorizontalPodAutoscaler()
				ns.HorizontalPodAutoscalers[name] = hpa
				ei = hpa
				f = true
			}
		default:
			var crd Entities
			if crd, ok = ns.CRDs[kind]; force && !ok {
				crd = make(Entities)
				ns.CRDs[kind] = crd
				ok = true
			}
			if ok {
				ei, f = crd[name]
				if force && !f {
					e := newEntity()
					crd[name] = e
					ei = e
					f = true
				}
			}
		}
	} else {
		if force && kind == NamespaceKey {
			ns := newNamespace()
			namespaces[namespace] = ns
			ei = ns
			f = true
		}
	}
	return
}

func GetContainer(p *Pod, name string) (c *Container, f bool) {
	return GetOrCreateContainer(p, name, false)
}

func EnsureContainer(p *Pod, name string) (c *Container, f bool) {
	return GetOrCreateContainer(p, name, true)
}

func GetOrCreateContainer(p *Pod, name string, force bool) (c *Container, f bool) {
	if p != nil {
		c, f = p.Containers[name]
		if force && !f {
			c = newContainer()
			p.Containers[name] = c
			f = true
		}
	}
	return
}

func newEntity() *Entity {
	return &Entity{
		OwnerName:    &Labels{},
		OwnerKind:    &Labels{},
		CreationTime: &Labels{},
		DeletionTime: &Labels{},
		Generation:   &Labels{},
		LabelMap:     make(LabelMap),
	}
}

func newConditionManagement() *ConditionManagement {
	return &ConditionManagement{
		Conditions:    make(LabelMap),
		RawConditions: make(map[string]*Condition),
	}
}

func newReplicationManagement() *ReplicationManagement {
	return &ReplicationManagement{DesiredReplicas: &Labels{}, CurrentReplicas: &Labels{}}
}

func newGenerationManagement() *GenerationManagement {
	return &GenerationManagement{CurrentGeneration: &Labels{}}
}

func newStandardRollingUpdateStrategy() *StandardRollingUpdateStrategy {
	return &StandardRollingUpdateStrategy{MaxSurge: &Labels{}, MaxUnavailable: &Labels{}}
}

func newNamespace() *Namespace {
	return &Namespace{
		Entity:                   newEntity(),
		CRDs:                     make(map[string]Entities),
		HorizontalPodAutoscalers: make(map[string]*HorizontalPodAutoscaler),
		CronJobs:                 make(map[string]*CronJob),
		Jobs:                     make(map[string]*Job),
		Deployments:              make(map[string]*Deployment),
		ReplicaSets:              make(map[string]*ReplicaSet),
		ReplicationControllers:   make(map[string]*ReplicationController),
		StatefulSets:             make(map[string]*StatefulSet),
		DaemonSets:               make(map[string]*DaemonSet),
		Pods:                     make(map[string]*Pod),
	}
}

func newPod() *Pod {
	return &Pod{
		Entity:              newEntity(),
		ConditionManagement: newConditionManagement(),
		Containers:          make(map[string]*Container),
	}
}

func newReplicaSet() *ReplicaSet {
	return &ReplicaSet{
		Entity:                newEntity(),
		ReplicationManagement: newReplicationManagement(),
	}
}

func newReplicationController() *ReplicationController {
	return &ReplicationController{
		Entity:                newEntity(),
		ReplicationManagement: newReplicationManagement(),
	}
}

func newDeployment() *Deployment {
	return &Deployment{
		Entity:                        newEntity(),
		ReplicationManagement:         newReplicationManagement(),
		GenerationManagement:          newGenerationManagement(),
		StandardRollingUpdateStrategy: newStandardRollingUpdateStrategy(),
		ConditionManagement:           newConditionManagement(),
		Paused:                        &Labels{},
	}
}

func newDaemonSet() *DaemonSet {
	return &DaemonSet{
		Entity:                        newEntity(),
		ReplicationManagement:         newReplicationManagement(),
		GenerationManagement:          newGenerationManagement(),
		StandardRollingUpdateStrategy: newStandardRollingUpdateStrategy(),
	}
}

func newStatefulSet() *StatefulSet {
	return &StatefulSet{
		Entity:                newEntity(),
		ReplicationManagement: newReplicationManagement(),
		GenerationManagement:  newGenerationManagement(),
		Partition:             &Labels{},
	}
}

func newJob() *Job {
	return &Job{
		Entity:              newEntity(),
		ConditionManagement: newConditionManagement(),
		Completions:         &Labels{},
		Parallelism:         &Labels{},
		StartTime:           &Labels{},
		CompletionTime:      &Labels{},
	}
}

func newCronJob() *CronJob {
	return &CronJob{
		Entity:                     newEntity(),
		Schedule:                   &Labels{},
		ConcurrencyPolicy:          &Labels{},
		Suspend:                    &Labels{},
		SuccessfulJobsHistoryLimit: &Labels{},
		FailedJobsHistoryLimit:     &Labels{},
		Active:                     &Labels{},
		LastScheduledTime:          &Labels{},
		NextScheduledTime:          &Labels{},
	}
}

func newHorizontalPodAutoscaler() *HorizontalPodAutoscaler {
	return &HorizontalPodAutoscaler{
		Entity:                newEntity(),
		ReplicationManagement: newReplicationManagement(),
		ConditionManagement:   newConditionManagement(),
		TargetObjectKind:      &Labels{},
		TargetObjectName:      &Labels{},
		MetricName:            &Labels{},
		MetricTargetType:      &Labels{},
		MinReplicas:           &Labels{},
		MaxReplicas:           &Labels{},
	}
}

func newContainer() *Container {
	return &Container{
		CpuLimit:   &Labels{},
		CpuRequest: &Labels{},
		MemLimit:   &Labels{},
		MemRequest: &Labels{},
		PowerState: &Labels{},
		LabelMap:   make(LabelMap),
	}
}

func NewNode() *Node {
	return &Node{
		Entity:              newEntity(),
		ConditionManagement: newConditionManagement(),
		Roles:               make(LabelMap),
		NetSpeedBytesMap:    make(LabelMap),
		Capacity:            make(LabelMap),
		Allocatable:         make(LabelMap),
	}
}

func NewResourceQuota() *ResourceQuota {
	return &ResourceQuota{Entity: newEntity()}
}

func NewClusterResourceQuota() *ClusterResourceQuota {
	return &ClusterResourceQuota{
		Entity:        newEntity(),
		SelectorType:  &Labels{},
		SelectorKey:   &Labels{},
		SelectorValue: &Labels{},
	}
}
