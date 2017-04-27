package types

import (
	"encoding/json"
	"fmt"
	"net"
)

// IPAMConfig represents the IP related network configuration.
type IPAMConfig struct {
	Name   string
	Type   string    `json:"type"`
	Remote string    `json:"remote"`

	// CA root file
	CA	   string 	 `json:"ca"`
	Args   *IPAMArgs `json:"-"`
}

type IPAMArgs struct {
	CommonArgs
	PodID  string `json:"podid"`
	HostIP net.IP `json:"hostip"`
}

type Net struct {
	Name string      `json:"name"`
	IPAM *IPAMConfig `json:"ipam"`
}

// NewIPAMConfig creates a NetworkConfig from the given network name.
func LoadIPAMConfig(bytes []byte, args string) (*IPAMConfig, error) {
	n := Net{}
	if err := json.Unmarshal(bytes, &n); err != nil {
		return nil, err
	}

	if args != "" {
		n.IPAM.Args = &IPAMArgs{}
		err := LoadArgs(args, n.IPAM.Args)
		if err != nil {
			return nil, err
		}
	}

	if n.IPAM == nil {
		return nil, fmt.Errorf("IPAM config missing 'ipam' key")
	}

	// Copy net name into IPAM so not to drag Net struct around
	n.IPAM.Name = n.Name

	return n.IPAM, nil
}
