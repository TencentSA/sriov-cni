package main

import (
	"encoding/json"

	"github.com/containernetworking/cni/pkg/types"

	tt "txipam/types"
)

func convertCniTypesToTxipamTypes(r *types.Result) (*tt.Result, error) {
	res := &tt.Result{}
	err := json.Unmarshal(r.Raw, res)
	return res, err
}
