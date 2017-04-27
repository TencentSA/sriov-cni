package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

const VfIrqPath = "/sys/class/net/%s/device/virtfn%d/msi_irqs"
const IrqAffinityPath = "/proc/irq/%d/smp_affinity"

func GetVfIrqs(master string, vf int) ([]int, error) {
	var irqs []int

	irqDir := fmt.Sprintf(VfIrqPath, master, vf)
	if _, err := os.Lstat(irqDir); err != nil {
		return nil, fmt.Errorf("failed to open %s: %v", irqDir, err)
	}

	infos, err := ioutil.ReadDir(irqDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %v", irqDir, err)
	}

	if len(infos) == 0 {
		return nil, fmt.Errorf("irqs quanity is zero")
	}

	for _, info := range infos {
		i, err := strconv.Atoi(info.Name())
		if err != nil {
			return nil, err
		}
		irqs = append(irqs, i)
	}
	return irqs, nil
}

func SetIrqCpuAffinity(irq, core int) error {

	irqAfPath := fmt.Sprintf(IrqAffinityPath, irq)
	if _, err := os.Lstat(irqAfPath); err != nil {
		return fmt.Errorf("failed to check %s: %v", irqAfPath, err)
	}

	if core >= 64 {
		return fmt.Errorf("CPU core value must be smaller than 64")
	}
	m := 1 << uint64(core)

	val := fmt.Sprintf("%x", m)

	if err := ioutil.WriteFile(irqAfPath, []byte(val), 0600); err != nil {
		return fmt.Errorf("failed to write affinity %v", err)
	}
	return nil
}

func CpuCoreToMask(cores string) (string, error) {
	var mask uint64 = 0
	cs := strings.Split(cores, ",")
	for _, c := range cs {
		i, err := strconv.Atoi(c)
		if err != nil {
			return "", err
		}
		mask = mask | (1 << uint64(i))
	}

	r := fmt.Sprintf("%x", mask)
	return r, nil
}

func SetRpsCpuAffinity(nic, cores string) error {
	queDir := fmt.Sprintf("/sys/class/net/%s/queues", nic)
	if _, err := os.Lstat(queDir); err != nil {
		return fmt.Errorf("failed to open %s: %v", queDir, err)
	}

	infos, err := ioutil.ReadDir(queDir)
	if err != nil {
		return fmt.Errorf("failed to read %s: %v", queDir, err)
	}

	mask, err := CpuCoreToMask(cores)
	if err != nil {
		return fmt.Errorf("calc rps cpu %s mask error: %v", cores, err)
	}
	for _, info := range infos {
		queName := info.Name()
		if strings.HasPrefix(queName, "rx-") {
			rpsCpu := fmt.Sprintf("/sys/class/net/%s/queues/%s/rps_cpus", nic, queName)
			if err := ioutil.WriteFile(rpsCpu, []byte(mask), 0644); err != nil {
				return fmt.Errorf("failed to write rps_cpus: %v", err)
			}
		}
	}
	return nil
}
