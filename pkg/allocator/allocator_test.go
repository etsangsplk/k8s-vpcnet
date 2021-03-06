package allocator

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"testing"

	"github.com/lstoll/k8s-vpcnet/pkg/config"
	"github.com/lstoll/k8s-vpcnet/pkg/nodestate"
)

var testMap = nodestate.ENIs{
	{
		EniID:       "eni-5d232b8d",
		Attached:    true,
		InterfaceIP: net.ParseIP("10.0.8.96"),
		CIDRBlock:   &config.IPNet{IP: net.ParseIP("10.0.0.0"), Mask: net.CIDRMask(20, 32)},
		Index:       1,
		IPs: []net.IP{
			net.ParseIP("10.0.10.32"),
			net.ParseIP("10.0.15.243"),
			net.ParseIP("10.0.15.36"),
		},
		MACAddress: "0a:3e:1f:4e:c6:d2",
	},
	{
		EniID:       "eni-5d232b8e",
		Attached:    true,
		InterfaceIP: net.ParseIP("10.0.8.97"),
		CIDRBlock:   &config.IPNet{IP: net.ParseIP("10.0.0.0"), Mask: net.CIDRMask(20, 32)},
		Index:       1,
		IPs: []net.IP{
			net.ParseIP("10.0.10.42"),
			net.ParseIP("10.0.15.233"),
			net.ParseIP("10.0.15.26"),
		},
		MACAddress: "0a:3e:1f:4e:c6:dd",
	},
}

func TestAllocator(t *testing.T) {
	workDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workDir)

	alloc, err := New(workDir + "state.json")
	if err != nil {
		t.Fatalf("Error creating allocator [%+v]", err)
	}

	if err := alloc.SetENIs(testMap); err != nil {
		t.Fatalf("Error setting ENIS's on allocator [%+v]", err)
	}

	if alloc.FreeAddressCount() != 6 {
		t.Errorf("Expected 6 free addresses, got %d", alloc.FreeAddressCount())
	}

	// try allocating and deallocating max addresses
	for i := 0; i < 6; i++ {
		_, err := alloc.Allocate(fmt.Sprintf("abc%d", i), fmt.Sprintf("pod-abc%d", i), "ns")
		if err != nil {
			t.Fatalf("Error getting address %d [%+v]", i, err)
		}
	}

	if alloc.FreeAddressCount() != 0 {
		t.Errorf("Expected 0 free addresses, got %d", alloc.FreeAddressCount())
	}

	_, err = alloc.Allocate("abc7", "pod-abc7", "ns")
	if err != ErrNoFreeIPs {
		t.Fatal("Expected ErrNoFreeIPs error allocating 7th address")
	}

	// Free 3 in the middle
	for i := 2; i < 5; i++ {
		err := alloc.ReleaseByContainer(fmt.Sprintf("abc%d", i))
		if err != nil {
			t.Fatalf("Error freeing address %d [%+v]", i, err)
		}
	}

	if alloc.FreeAddressCount() != 3 {
		t.Errorf("Expected 3 free addresses, got %d", alloc.FreeAddressCount())
	}

	// Allocate 3 more
	for i := 10; i < 13; i++ {
		_, err := alloc.Allocate(fmt.Sprintf("abc%d", i), fmt.Sprintf("pod-abc%d", i), "ns")
		if err != nil {
			t.Fatalf("Error getting address %d [%+v]", i, err)
		}
	}

	if alloc.FreeAddressCount() != 0 {
		t.Errorf("Expected 0 free addresses, got %d", alloc.FreeAddressCount())
	}

	// extra alloc should fail
	_, err = alloc.Allocate("abc17", "pod-abc17", "ns")
	if err == nil {
		t.Fatal("Expected error allocating extra address")
	}
}
