package datamodel

import "strings"

type ContainerType int

const (
	RegularContainer   = "Regular"
	InitContainer      = "Init"
	EphemeralContainer = "Ephemeral"
)

const (
	_ ContainerType = iota
	Regular
	Init
	Ephemeral
)

var com = map[string]ContainerType{
	strings.ToLower(RegularContainer):   Regular,
	strings.ToLower(InitContainer):      Init,
	strings.ToLower(EphemeralContainer): Ephemeral,
}

var rcom = map[ContainerType]string{
	Regular:   RegularContainer,
	Init:      InitContainer,
	Ephemeral: EphemeralContainer,
}

func ToContainerType(s string) (c ContainerType, f bool) {
	c, f = com[strings.ToLower(s)]
	return
}

func ToContainerTypeString(c ContainerType) (s string, f bool) {
	s, f = rcom[c]
	return
}
