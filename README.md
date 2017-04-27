# SR-IOV CNI plugin

If you do not know CNI. Please read [here](https://github.com/containernetworking/cni) at first.

NIC with [SR-IOV](http://blog.scottlowe.org/2009/12/02/what-is-sr-iov/) capabilities works by introducing the idea of physical functions (PFs) and virtual functions (VFs). 

PF is used by host.Each VFs can be treated as a separate physical NIC and assigned to one container, and configured with separate MAC, VLAN and IP, etc.

## Build

This plugin requires Go 1.5+ to build.

Go 1.5 users will need to set `GO15VENDOREXPERIMENT=1` to get vendored dependencies. This flag is set by default in 1.6.

```
#./build
```

## Enable SR-IOV

Given Intel ixgbe NIC on CentOS, Fedora or RHEL:

```
# vi /etc/modprobe.conf
options ixgbe max_vfs=8,8
```

## Network configuration reference

* `name` (string, required): the name of the network
* `type` (string, required): "sriov"
* `master` (string, required): name of the PF
* `ipam` (dictionary, required): IPAM configuration to be used for this network.

## Extra arguments

* `vf` (int, optional): VF index. This plugin will allocate a free VF if not assigned
* `vlan` (int, optional): VLAN ID for VF device
* `mac` (string, optional): mac address for VF device

## With `txipam`

* `PodID` (string, required): pod id
* `HostIP` (string, reqired): host IP
* `CORES` (string, optional): VF binding CPU core

## Usage

Given the following network configuration:

```
# cat > /etc/cni/net.d/10-mynet.conf <<EOF
{
    "name": "mynet",
    "type": "sriov",
    "master": "eth1",
    "ipam": {
        "type": "txipam",
        "remote": "https://127.0.0.1:10000",
        "ca": "/etc/pki/ca-trust/source/anchors/ca.crt"
    }
}
EOF
```

Add container to network:

```sh
# CNI_PATH=`pwd`/bin
# CNI_PATH=$CNI_PATH CNI_COMMAND=ADD CNI_NETNS=/var/run/netns/1234 CNI_CONTAINERID=1234 CNI_IFNAME=eth1 CNI_ARGS="IgnoreUnknown=1;PodID=1234;HostIP=10.55.206.20;CORES=1,2,3,4" bin/sriov < /etc/cni/net.d/txnet.conf
{
    "ip4": {
        "ip": "10.55.206.46/26",
        "gateway": "10.55.206.1"
    },
    "dns": {}
}

# ip netns exec 1234 ip a
1: lo: <LOOPBACK> mtu 65536 qdisc noop state DOWN
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
13: eth1: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc mq state UP qlen 1000
    link/ether aa:bb:cc:dd:ee:ff brd ff:ff:ff:ff:ff:ff
    inet 10.55.206.46/26 scope global eth1
       valid_lft forever preferred_lft forever
    inet6 fe80::a8bb:ccff:fedd:eeff/64 scope link
       valid_lft forever preferred_lft forever

# ls /sys/class/net/eth1/device/virtfn1/msi_irqs/
174  175
# cat /proc/irq/174/smp_affinity
0000,00000002
# cat /proc/irq/175/smp_affinity
0000,00000004
```

Remove container from network:

```sh
# CNI_PATH=$CNI_PATH CNI_COMMAND=DEL CNI_NETNS=/var/run/netns/1234 CNI_CONTAINERID=1234 CNI_IFNAME=eth1 CNI_ARGS="PodID=1234;HostIP=10.55.206.20" bin/sriov < /etc/cni/net.d/txnet.conf
```

[More info](https://github.com/containernetworking/cni/pull/259).
