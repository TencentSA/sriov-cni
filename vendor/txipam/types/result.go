package types

import (
	"encoding/json"
	"os"

	"github.com/containernetworking/cni/pkg/types"
)

type ExtraInfo struct {
    VLAN int    `json:"vlan,omitempty"`
    MAC  string `json:"mac,omitempty"`
    VF   *int   `json:"vf,omitempty"`
}

type Result struct {
	types.Result
	Extra *ExtraInfo `json:"extra,omitempty"`
}

func (r *Result) Print() error {

	data, err := json.MarshalIndent(r, "", "    ")
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(data)
	return err
}
