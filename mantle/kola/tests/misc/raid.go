// Copyright 2017 CoreOS, Inc.
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
	"github.com/coreos/mantle/kola/cluster"
	"github.com/coreos/mantle/kola/register"
	"github.com/coreos/mantle/kola/tests/util"
	"github.com/coreos/mantle/platform"
	"github.com/coreos/mantle/platform/conf"
)

var (
	raidRootUserData = conf.ContainerLinuxConfig(`storage:
  disks:
    - device: "/dev/disk/by-id/virtio-disk1"
      wipe_table: true
      partitions:
       - label: root1
         number: 1
         size: 256MiB
         type_guid: be9067b9-ea49-4f15-b4f6-f36f8c9e1818
       - label: root2
         number: 2
         size: 256MiB
         type_guid: be9067b9-ea49-4f15-b4f6-f36f8c9e1818
  raid:
    - name: "rootarray"
      level: "raid1"
      devices:
        - "/dev/disk/by-partlabel/root1"
        - "/dev/disk/by-partlabel/root2"
  filesystems:
    - name: "ROOT"
      mount:
        device: "/dev/md/rootarray"
        format: "ext4"
        label: ROOT
    - name: "NOT_ROOT"
      mount:
        device: "/dev/disk/by-id/virtio-primary-disk-part9"
        format: "ext4"
        label: wasteland
        wipe_filesystem: true`)
)

func init() {
	register.RegisterTest(&register.Test{
		// This test needs additional disks which is only supported on qemu since Ignition
		// does not support deleting partitions without wiping the partition table and the
		// disk doesn't have room for new partitions.
		// TODO(ajeddeloh): change this to delete partition 9 and replace it with 9 and 10
		// once Ignition supports it.
		Run:         RootOnRaid,
		ClusterSize: 0,
		Platforms:   []string{"qemu"},
		Name:        "cl.disk.raid.root",
		Distros:     []string{"cl"},
	})
	register.RegisterTest(&register.Test{
		Run:         DataOnRaid,
		ClusterSize: 1,
		Name:        "cl.disk.raid.data",
		UserData: conf.ContainerLinuxConfig(`storage:
  raid:
    - name: "DATA"
      level: "raid1"
      devices:
        - "/dev/disk/by-partlabel/OEM-CONFIG"
        - "/dev/disk/by-partlabel/USR-B"
  filesystems:
    - name: "DATA"
      mount:
        device: "/dev/md/DATA"
        format: "ext4"
        label: DATA
systemd:
  units:
    - name: "var-lib-data.mount"
      enable: true
      contents: |
          [Mount]
          What=/dev/md/DATA
          Where=/var/lib/data
          Type=ext4
          
          [Install]
          WantedBy=local-fs.target`),
		Distros: []string{"cl"},
	})
}

func RootOnRaid(c cluster.TestCluster) {
	var m platform.Machine
	var err error
	options := platform.MachineOptions{
		AdditionalDisks: []string{"520M"},
	}
	m, err = c.NewMachineWithOptions(raidRootUserData, options)
	if err != nil {
		c.Fatal(err)
	}

	util.CheckIfMountpointIsRaid(c, m, "/")

	// reboot it to make sure it comes up again
	err = m.Reboot()
	if err != nil {
		c.Fatalf("could not reboot machine: %v", err)
	}

	util.CheckIfMountpointIsRaid(c, m, "/")
}

func DataOnRaid(c cluster.TestCluster) {
	m := c.Machines()[0]

	util.CheckIfMountpointIsRaid(c, m, "/var/lib/data")

	// reboot it to make sure it comes up again
	err := m.Reboot()
	if err != nil {
		c.Fatalf("could not reboot machine: %v", err)
	}

	util.CheckIfMountpointIsRaid(c, m, "/var/lib/data")
}
