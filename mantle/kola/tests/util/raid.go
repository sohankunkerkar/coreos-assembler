// Copyright 2018 Red Hat, Inc.
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

package util

import (
	"encoding/json"

	"github.com/coreos/mantle/kola/cluster"
	"github.com/coreos/mantle/platform"
)

type lsblkOutput struct {
	Blockdevices []blockdevice `json:"blockdevices"`
}

type blockdevice struct {
	Name       string        `json:"name"`
	Type       string        `json:"type"`
	Mountpoint *string       `json:"mountpoint"`
	Children   []blockdevice `json:"children"`
}

// CheckIfMountpointIsRaid will check if a given machine has a device of type
// raid1 mounted at the given mountpoint. If it does not, the test is failed.
func CheckIfMountpointIsRaid(c cluster.TestCluster, m platform.Machine, mountpoint string) {
	output := c.MustSSH(m, "lsblk --json")

	l := lsblkOutput{}
	err := json.Unmarshal(output, &l)
	if err != nil {
		c.Fatalf("couldn't unmarshal lsblk output: %v", err)
	}

	foundRoot := checkIfMountpointIsRaidWalker(c, l.Blockdevices, mountpoint)
	if !foundRoot {
		c.Fatalf("didn't find %v mountpoint in lsblk output", mountpoint)
	}
}

// checkIfMountpointIsRaidWalker will iterate over bs and recurse into its
// children, looking for a device mounted at / with type raid1. true is returned
// if such a device is found. The test is failed if a device of a different type
// is found to be mounted at /.
func checkIfMountpointIsRaidWalker(c cluster.TestCluster, bs []blockdevice, mountpoint string) bool {
	for _, b := range bs {
		if b.Mountpoint != nil && *b.Mountpoint == mountpoint {
			if b.Type != "raid1" {
				c.Fatalf("device %q is mounted at %q with type %q (was expecting raid1)", b.Name, mountpoint, b.Type)
			}
			return true
		}
		foundRoot := checkIfMountpointIsRaidWalker(c, b.Children, mountpoint)
		if foundRoot {
			return true
		}
	}
	return false
}
