package util

import (
	"testing"
)

func TestSetIrqCpuAffinity(t *testing.T) {
	if err := SetIrqCpuAffinity(51, 1); err != nil {
		t.Errorf("Set cpu affinity error: %v", err)
	}
}

func TestCpuCoreToMask(t *testing.T) {
	cores := "1,2,3,4"
	r, err := CpuCoreToMask(cores)
	if err != nil || r != "1e" {
		t.Errorf("%v", err)
	}
}

func TestSetRpsCpuAffinity(t *testing.T) {
	err := SetRpsCpuAffinity("eth1", "1,2,3,4")
	if err != nil {
		t.Errorf("%v", err)
	}
}
