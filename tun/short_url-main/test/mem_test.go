package test

import (
	"testing"

	"github.com/shirou/gopsutil/v3/mem"
)

func TestMem_VirtualMemory(t *testing.T) {
	v, err := mem.VirtualMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("v:", v)
	t.Log("v.UsedPercent:", v.UsedPercent, "%")
	t.Log("v.Total(B):", v.Total)
	t.Log("v.Total(GB):", float64(v.Total)/1024/1024/1024)
	t.Log("v.Free:", v.Free)
	t.Log("v.Used:", v.Used)
	t.Log("v.Shared:", v.Shared)
	t.Log("v.Buffers:", v.Buffers)
	t.Log("v.Cached:", v.Cached)
	t.Log("v.Active:", v.Active)
	t.Log("v.Inactive:", v.Inactive)

	t.Log("calculate:", int(float64(v.Total)/100*3/256))
}
