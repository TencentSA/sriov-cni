package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/containernetworking/cni/pkg/ipam"
	"github.com/containernetworking/cni/pkg/ns"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/vishvananda/netlink"

	. "github.com/hustcat/sriov-cni/config"
)

func init() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
}

func loadConf(bytes []byte, args string) (*SriovConf, error) {
	n := &NetConf{}
	a := &NetArgs{}

	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, fmt.Errorf("failed to load netconf: %v", err)
	}
	if n.Master == "" {
		return nil, fmt.Errorf(`"master" field is required. It specifies the host interface name to virtualize`)
	}

	if args != "" {
		err := LoadSriovArgs(args, a)
		if err != nil {
			return nil, fmt.Errorf("failed to parse args: %v", err)
		}
	}

	return &SriovConf{
		Net:  n,
		Args: a,
	}, nil
}

func setupVF(conf *SriovConf, ifName string, netns ns.NetNS) error {
	var err error
	var vfDevName string

	vfIdx := 0
	masterName := conf.Net.Master
	args := conf.Args

	if args.VF != 0 {
		vfIdx = args.VF
	} else {
		// alloc a free virtual function
		if vfIdx, vfDevName, err = allocFreeVF(masterName); err != nil {
			return err
		}
	}

	m, err := netlink.LinkByName(masterName)
	if err != nil {
		return fmt.Errorf("failed to lookup master %q: %v", masterName, err)
	}

	vfDev, err := netlink.LinkByName(vfDevName)
	if err != nil {
		return fmt.Errorf("failed to lookup vf device %q: %v", vfDevName, err)
	}

	// set hardware address
	if args.MAC != "" {
		macAddr, err := net.ParseMAC(args.MAC)
		if err != nil {
			return err
		}
		if err = netlink.LinkSetVfHardwareAddr(m, vfIdx, macAddr); err != nil {
			return fmt.Errorf("failed to set vf %d macaddress: %v", vfIdx, err)
		}
	}

	if args.VLAN != 0 {
		if err = netlink.LinkSetVfVlan(m, vfIdx, args.VLAN); err != nil {
			return fmt.Errorf("failed to set vf %d vlan: %v", vfIdx, err)
		}
	}

	if err = netlink.LinkSetUp(vfDev); err != nil {
		return fmt.Errorf("failed to setup vf %d device: %v", vfIdx, err)
	}

	// move VF device to ns
	if err = netlink.LinkSetNsFd(vfDev, int(netns.Fd())); err != nil {
		return fmt.Errorf("failed to move vf %d to netns: %v", vfIdx, err)
	}

	return netns.Do(func(_ ns.NetNS) error {
		err := renameLink(vfDevName, ifName)
		if err != nil {
			return fmt.Errorf("failed to rename vf %d device %q to %q: %v", vfIdx, vfDevName, ifName, err)
		}
		return nil
	})
}

func releaseVF(conf *SriovConf, ifName string, netns ns.NetNS) error {
	vfIdx := 0
	args := conf.Args
	if args.VF != 0 {
		vfIdx = args.VF
	}

	initns, err := ns.GetCurrentNS()
	if err != nil {
		return fmt.Errorf("failed to get init netns: %v", err)
	}

	if err = netns.Set(); err != nil {
		return fmt.Errorf("failed to enter netns %q: %v", netns, err)
	}

	// get VF device
	vfDev, err := netlink.LinkByName(ifName)
	if err != nil {
		return fmt.Errorf("failed to lookup vf %d device %q: %v", vfIdx, ifName, err)
	}

	// device name in init netns
	index := vfDev.Attrs().Index
	devName := fmt.Sprintf("dev%d", index)

	// shutdown VF device
	if err = netlink.LinkSetDown(vfDev); err != nil {
		return fmt.Errorf("failed to down vf %d device: %v", vfIdx, err)
	}

	// rename VF device
	err = renameLink(ifName, devName)
	if err != nil {
		return fmt.Errorf("failed to rename vf %d evice %q to %q: %v", vfIdx, ifName, devName, err)
	}

	// move VF device to init netns
	if err = netlink.LinkSetNsFd(vfDev, int(initns.Fd())); err != nil {
		return fmt.Errorf("failed to move vf %d to init netns: %v", vfIdx, err)
	}

	return nil
}

func cmdAdd(args *skel.CmdArgs) error {
	n, err := loadConf(args.StdinData, args.Args)
	if err != nil {
		return err
	}

	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		return fmt.Errorf("failed to open netns %q: %v", args.Netns, err)
	}
	defer netns.Close()

	if err = setupVF(n, args.IfName, netns); err != nil {
		return err
	}

	if err := resetCniArgsForIPAM(n.Args); err != nil {
		return err
	}

	// run the IPAM plugin and get back the config to apply
	result, err := ipam.ExecAdd(n.Net.IPAM.Type, args.StdinData)
	if err != nil {
		return err
	}
	if result.IP4 == nil {
		return errors.New("IPAM plugin returned missing IPv4 config")
	}

	err = netns.Do(func(_ ns.NetNS) error {
		return ipam.ConfigureIface(args.IfName, result)
	})
	if err != nil {
		return err
	}

	result.DNS = n.Net.DNS
	return result.Print()
}

func cmdDel(args *skel.CmdArgs) error {
	n, err := loadConf(args.StdinData, args.Args)
	if err != nil {
		return err
	}

	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		return fmt.Errorf("failed to open netns %q: %v", netns, err)
	}
	defer netns.Close()

	if err = releaseVF(n, args.IfName, netns); err != nil {
		return err
	}

	if err := resetCniArgsForIPAM(n.Args); err != nil {
		return err
	}

	err = ipam.ExecDel(n.Net.IPAM.Type, args.StdinData)
	if err != nil {
		return err
	}

	return nil
}

func renameLink(curName, newName string) error {
	link, err := netlink.LinkByName(curName)
	if err != nil {
		return err
	}

	return netlink.LinkSetName(link, newName)
}

func resetCniArgsForIPAM(args *NetArgs) error {
	argVal := ""

	if args.IP != nil {
		argVal = fmt.Sprintf("IP=%s", args.IP.String())
	}
	if err := os.Setenv("CNI_ARGS", argVal); err != nil {
		return fmt.Errorf("reset cni args for ipam failed: %v", err)
	}

	return nil
}

func allocFreeVF(master string) (int, string, error) {
	vfIdx := -1
	devName := ""

	sriovFile := fmt.Sprintf("/sys/class/net/%s/device/sriov_numvfs", master)
	if _, err := os.Lstat(sriovFile); err != nil {
		return -1, "", fmt.Errorf("failed to open the sriov_numfs of device %q: %v", master, err)
	}

	data, err := ioutil.ReadFile(sriovFile)
	if err != nil {
		return -1, "", fmt.Errorf("failed to read the sriov_numfs of device %q: %v", master, err)
	}

	if len(data) == 0 {
		return -1, "", fmt.Errorf("no data in the file %q", sriovFile)
	}

	sriovNumfs := strings.TrimSpace(string(data))
	vfTotal, err := strconv.Atoi(sriovNumfs)
	if err != nil {
		return -1, "", fmt.Errorf("failed to convert sriov_numfs(byte value) to int of device %q: %v", master, err)
	}

	if vfTotal <= 0 {
		return -1, "", fmt.Errorf("no virtual function in the device %q: %v", master)
	}

	for vf := 0; vf < vfTotal; vf++ {
		devName, err = getVFDeviceName(master, vf)

		// got a free vf
		if err == nil {
			vfIdx = vf
			break
		}
	}

	if vfIdx == -1 {
		return -1, "", fmt.Errorf("can not get a free virtual function in directory %s", master)
	}
	return vfIdx, devName, nil
}

func getVFDeviceName(master string, vf int) (string, error) {
	vfDir := fmt.Sprintf("/sys/class/net/%s/device/virtfn%d/net", master, vf)
	if _, err := os.Lstat(vfDir); err != nil {
		return "", fmt.Errorf("failed to open the virtfn%d dir of the device %q: %v", vf, master, err)
	}

	infos, err := ioutil.ReadDir(vfDir)
	if err != nil {
		return "", fmt.Errorf("failed to read the virtfn%d dir of the device %q: %v", vf, master, err)
	}

	if len(infos) != 1 {
		return "", fmt.Errorf("no network device in directory %s", vfDir)
	}
	return infos[0].Name(), nil
}

func main() {
	skel.PluginMain(cmdAdd, cmdDel)
}
