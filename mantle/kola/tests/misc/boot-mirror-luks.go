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
	"regexp"
	"strings"

	"github.com/coreos/mantle/kola"
	"github.com/coreos/mantle/kola/cluster"
	"github.com/coreos/mantle/kola/register"
	"github.com/coreos/mantle/platform"
	"github.com/coreos/mantle/platform/conf"
	"github.com/coreos/mantle/platform/machine/unprivqemu"
	"github.com/coreos/mantle/system"
)

var (
	/* The FCCT config used for generating this ignition config:
	variant: fcos
	version: 1.3.0
	boot_device:
	  luks:
	    tpm2: true
	  mirror:
	    devices:
		  - /dev/sda
		  - /dev/sdb
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
	  raid:
	    - name: md-boot
		  level: raid1
		  devices:
		    - /dev/disk/by-partlabel/boot-1
		    - /dev/disk/by-partlabel/boot-2
	*/
	bootmirrorluks = conf.Ignition(`{
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
			}
		  ],
		  "filesystems": [
			{
			  "device": "/dev/disk/by-partlabel/esp-1",
			  "format": "vfat",
			  "label": "esp1",
			  "wipeFilesystem": true
			},
			{
				"device": "/dev/disk/by-partlabel/esp-2",
				"format": "vfat",
				"label": "esp2",
				"wipeFilesystem": true
			},
			{
			  "device": "/dev/md/md-boot",
			  "format": "ext4",
			  "label": "boot",
			  "wipeFilesystem": true
			},
			{
			  "device": "/dev/mapper/root",
			  "format": "xfs",
			  "label": "root",
			  "wipeFilesystem": true
			}
		  ],
		  "luks": [
			{
			  "clevis": {
				"tpm2": true
			  },
			  "device": "/dev/md/md-root",
			  "label": "luks-root",
			  "name": "root",
			  "wipeVolume": true
			}
		  ],
		  "raid": [
			{
			  "devices": [
				"/dev/disk/by-partlabel/boot-1",
				"/dev/disk/by-partlabel/boot-2"
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
				"/dev/disk/by-partlabel/root-2"
			  ],
			  "level": "raid1",
			  "name": "md-root"
			}
		  ]
		}
	  }`)
)

func mustMatch(c cluster.TestCluster, r string, output []byte) {
	m, err := regexp.Match(r, output)
	if err != nil {
		c.Fatalf("Failed to match regexp %s: %v", r, err)
	}
	if !m {
		c.Fatalf("Regexp %s did not match text: %s", r, output)
	}
}

func mustNotMatch(c cluster.TestCluster, r string, output []byte) {
	m, err := regexp.Match(r, output)
	if err != nil {
		c.Fatalf("Failed to match regexp %s: %v", r, err)
	}
	if m {
		c.Fatalf("Regexp %s matched text: %s", r, output)
	}
}

func init() {
	register.RegisterTest(&register.Test{
		Run:                  runBootMirrorLUKSTest,
		ClusterSize:          0,
		Name:                 `coreos.boot-mirror.luks`,
		Platforms:            []string{"qemu-unpriv"},
		ExcludeArchitectures: []string{"s390x"}, // no TPM support for s390x in qemu
		Tags:                 []string{"boot-mirror", "luks", "raid1", "tpm2", kola.NeedsInternetTag},
	})
}

// runBootMirrorLUKSTest verifies if the boot-mirror+LUKS RAID1
// flow works properly in both BIOS and UEFI modes.
func runBootMirrorLUKSTest(c cluster.TestCluster) {
	var m platform.Machine
	var err error
	options := platform.QemuMachineOptions{
		MachineOptions: platform.MachineOptions{
			AdditionalDisks: []string{"5120M"},
			MinMemory:       4096,
		},
	}
	m, err = c.Cluster.(*unprivqemu.Cluster).NewMachineWithQemuOptions(bootmirrorluks, options)
	if err != nil {
		c.Fatal(err)
	}
	luksSanityTest(c, m, true)
	// Check for root
	c.MustSSH(m, "sudo mdadm -D /dev/md/md-root")
	// Check for boot
	checkIfMountpointIsRaid(c, m, "/boot")
	fsTypeForBoot := c.MustSSH(m, "findmnt -nvr /boot -o FSTYPE")
	if strings.Compare(string(fsTypeForBoot), "ext4") != 0 {
		c.Fatalf("didn't match fstype for boot")
	}
	// Check that growpart didn't run
	c.MustSSH(m, "if [ -e /run/coreos-growpart.stamp ]; then exit 1; fi")

	if err := m.(platform.QEMUMachine).RemovePrimaryBlockDevice(); err != nil {
		c.Fatalf("failed to delete the first boot disk: %v", err)
	}
	err = m.Reboot()
	if err != nil {
		c.Fatalf("Failed to reboot the machine: %v", err)
	}
	// Check if there's only one device in the active raid
	c.MustSSH(m, "sudo mdadm --detail /dev/md/md-root")
	// Re-check luks device after rebooting a machine
	luksSanityTest(c, m, true)
	c.MustSSH(m, "grep root=UUID= /proc/cmdline")
	c.MustSSH(m, "grep rd.md.uuid= /proc/cmdline")
}

func luksSanityTest(c cluster.TestCluster, m platform.Machine, tpm2 bool) {
	rootPart := "/dev/md/md-root"
	// hacky,  but needed for s390x because of gpt issue with naming on big endian systems: https://bugzilla.redhat.com/show_bug.cgi?id=1899990
	if system.RpmArch() == "s390x" {
		rootPart = "/dev/disk/by-id/virtio-primary-disk-part4"
	}

	luksDump := c.MustSSH(m, "sudo cryptsetup luksDump "+rootPart)
	// Yes, some hacky regexps.  There is luksDump --debug-json but we'd have to massage the JSON
	// out of other debug output and it's not clear to me it's going to be more stable.
	// We're just going for a basic sanity check here.
	mustMatch(c, "Cipher: *aes", luksDump)
	mustNotMatch(c, "Cipher: *cipher_null-ecb", luksDump)
	mustMatch(c, "0: *clevis", luksDump)
	mustNotMatch(c, "9: *coreos", luksDump)

	s := c.MustSSH(m, "sudo clevis luks list -d "+rootPart)
	if tpm2 {
		mustMatch(c, "tpm2", s)
	}
	err := m.Reboot()
	if err != nil {
		c.Fatalf("Failed to reboot the machine: %v", err)
	}
	luksDump = c.MustSSH(m, "sudo cryptsetup luksDump "+rootPart)
	mustMatch(c, "Cipher: *aes", luksDump)
}
