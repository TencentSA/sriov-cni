package main

import (
	"github.com/containernetworking/cni/pkg/skel"
)

func main() {
	skel.PluginMain(cmdAdd, cmdDel)
}
