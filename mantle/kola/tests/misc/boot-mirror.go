// Copyright 2020 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package misc

import (
	"strings"

	"github.com/coreos/mantle/kola/cluster"
	"github.com/coreos/mantle/kola/register"
	"github.com/coreos/mantle/platform"
	"github.com/coreos/mantle/platform/conf"
	"github.com/coreos/mantle/platform/machine/unprivqemu"
)

var (
	/* The FCCT config used for generating this ignition config:
	variant: fcos
	version: 1.3.0
	boot_device:
	  mirror:
	    devices:
		  - /dev/sda
		  - /dev/sdb
		  - /dev/sdc
	storage:
	  disks:
	    - device: /dev/sda
		  partitions:
		    - label: root-1
			  size_mib: 5120
		    - label: boot-1
	    - device: /dev/sdb
		  partitions:
		    - label: root-2
			  size_mib: 5120
			- label: boot-2
	    - device: /dev/sdc
		  partitions:
		    - label: root-3
			  size_mib: 5120
		    - label: boot-3
	  raid:
	    - name: md-boot
		  level: raid1
		  devices:
		    - /dev/disk/by-partlabel/boot-1
			- /dev/disk/by-partlabel/boot-2
			- /dev/disk/by-partlabel/boot-3
	*/
	bootmirror = conf.Ignition(`{
		"ignition": {
			"version": "3.2.0"
		},
		"storage": {
			"disks": [
			  {
				"device": "/dev/vda",
				"partitions": [
				  {
					"label": "bios-1",
					"sizeMiB": 1,
					"typeGuid": "21686148-6449-6E6F-744E-656564454649"
				  },
				  {
					"label": "esp-1",
					"sizeMiB": 127,
					"typeGuid": "C12A7328-F81F-11D2-BA4B-00A0C93EC93B"
				  },
				  {
					"label": "boot-1",
					"sizeMiB": 384
				  },
				  {
					"label": "root-1"
				  }
				],
				"wipeTable": true
			  },
			  {
				"device": "/dev/vdb",
				"partitions": [
				  {
					"label": "bios-2",
					"sizeMiB": 1,
					"typeGuid": "21686148-6449-6E6F-744E-656564454649"
				  },
				  {
					"label": "esp-2",
					"sizeMiB": 127,
					"typeGuid": "C12A7328-F81F-11D2-BA4B-00A0C93EC93B"
				  },
				  {
					"label": "boot-2",
					"sizeMiB": 384
				  },
				  {
					"label": "root-2"
				  }
				],
				"wipeTable": true
			  },
			  {
				"device": "/dev/vdc",
				"partitions": [
				  {
					"label": "bios-3",
					"sizeMiB": 1,
					"typeGuid": "21686148-6449-6E6F-744E-656564454649"
				  },
				  {
					"label": "esp-3",
					"sizeMiB": 127,
					"typeGuid": "C12A7328-F81F-11D2-BA4B-00A0C93EC93B"
				  },
				  {
					"label": "boot-3",
					"sizeMiB": 384
				  },
				  {
					"label": "root-3"
				  }
				],
				"wipeTable": true
			  }
			],
			"filesystems": [
			 { 
				"device": "/dev/disk/by-partlabel/esp-1",
				"format": "vfat",
				"label": "esp-1",
				"wipeFilesystem": true
			  },
			  {
				"device": "/dev/disk/by-partlabel/esp-2",
				"format": "vfat",
				"label": "esp-2",
				"wipeFilesystem": true
			  },
			  {
				"device": "/dev/disk/by-partlabel/esp-3",
				"format": "vfat",
				"label": "esp-3",
				"wipeFilesystem": true
			  },
			  {
				"device": "/dev/md/md-boot",
				"format": "ext4",
				"label": "boot",
				"wipeFilesystem": true
			  },
			  {
				"device": "/dev/md/md-root",
				"format": "xfs",
				"label": "root",
				"wipeFilesystem": true
			  }
			],
			"raid": [
			  {
				"devices": [
				  "/dev/disk/by-partlabel/boot-1",
				  "/dev/disk/by-partlabel/boot-2",
				  "/dev/disk/by-partlabel/boot-3"
				],
				"level": "raid1",
				"name": "md-boot",
				"options": [
				  "--metadata=1.0"
				]
			  },
			  {
				"devices": [
				  "/dev/disk/by-partlabel/root-1",
				  "/dev/disk/by-partlabel/root-2",
				  "/dev/disk/by-partlabel/root-3"
				],
				"level": "raid1",
				"name": "md-root"
			  }
			]
		  }
	}`)
)

func init() {
	register.RegisterTest(&register.Test{
		Run:         runBootMirrorTest,
		ClusterSize: 0,
		Name:        `coreos.boot-mirror`,
		Platforms:   []string{"qemu-unpriv"},
		Tags:        []string{"boot-mirror", "raid1"},
	})
}

// runBootMirrorTest verifies if the boot-mirror RAID1
// flow works properly in both BIOS and UEFI modes.
func runBootMirrorTest(c cluster.TestCluster) {
	var m platform.Machine
	var err error
	options := platform.QemuMachineOptions{
		MachineOptions: platform.MachineOptions{
			AdditionalDisks: []string{"5120M", "5120M"},
			MinMemory:       4096,
		},
	}
	m, err = c.Cluster.(*unprivqemu.Cluster).NewMachineWithQemuOptions(bootmirror, options)
	if err != nil {
		c.Fatal(err)
	}
	// Check for root
	checkIfMountpointIsRaid(c, m, "/sysroot")
	fsTypeForRoot := c.MustSSH(m, "findmnt -nvr /sysroot -o FSTYPE")
	if strings.Compare(string(fsTypeForRoot), "xfs") != 0 {
		c.Fatalf("didn't match fstype for root")
	}
	// Check for boot
	checkIfMountpointIsRaid(c, m, "/boot")
	fsTypeForBoot := c.MustSSH(m, "findmnt -nvr /boot -o FSTYPE")
	if strings.Compare(string(fsTypeForBoot), "ext4") != 0 {
		c.Fatalf("didn't match fstype for boot")
	}
	// Check that growpart didn't run
	c.MustSSH(m, "if [ -e /run/coreos-growpart.stamp ]; then exit 1; fi")

	if err = m.(platform.QEMUMachine).RemovePrimaryBlockDevice(); err != nil {
		c.Fatalf("failed to delete the first boot disk: %v", err)
	}
	err = m.Reboot()
	if err != nil {
		c.Fatalf("Failed to reboot the machine: %v", err)
	}
	// Check if there's only one device in the active raid
	c.MustSSH(m, "sudo mdadm --detail /dev/md/md-root")
	c.MustSSH(m, "grep root=UUID= /proc/cmdline")
	c.MustSSH(m, "grep rd.md.uuid= /proc/cmdline")
}
