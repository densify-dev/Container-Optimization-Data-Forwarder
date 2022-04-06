package datamodel

import "strings"

type ReplicationStatusType int

const (
	ReplicasKey     = "replicas"
	CurrentReplicas = "Current"
	DesiredReplicas = "Desired"
)

const (
	_ ReplicationStatusType = iota
	Current
	Desired
)

var rm = map[string]ReplicationStatusType{
	strings.ToLower(CurrentReplicas): Current,
	strings.ToLower(DesiredReplicas): Desired,
}

var rrm = map[ReplicationStatusType]string{
	Current: CurrentReplicas,
	Desired: DesiredReplicas,
}

func ReplicasType(s string) (r ReplicationStatusType, f bool) {
	r, f = rm[strings.ToLower(s)]
	return
}

func ReplicasString(r ReplicationStatusType) (s string, f bool) {
	s, f = rrm[r]
	return
}

type ReplicationStatus struct {
	Type     ReplicationStatusType
	Replicas int
}
