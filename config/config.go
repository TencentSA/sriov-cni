package config

import (
	"github.com/containernetworking/cni/pkg/types"
)

type NetConf struct {
	types.NetConf
	Master string `json:"master"`
}

type NetArgs struct {
	types.CommonArgs
	VF    UnmarshallableInt          `json:"vf,omitempty"`
	VLAN  UnmarshallableInt          `json:"vlan,omitempty"`
	MAC   types.UnmarshallableString `json:"mac,omitempty"`
	CORES types.UnmarshallableString `json:"cores,omitempty"`
}

type SriovConf struct {
	Net  *NetConf
	Args *NetArgs
}
