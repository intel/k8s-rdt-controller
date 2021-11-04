/*
Copyright 2021 Intel Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package agent

import (
	"context"
	"io/ioutil"
	"os"
	"strings"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"

	"k8s.io/klog"
)

const (
	rdtLabel   = "RDT"
	hwdrcLabel = "HWDRC"
)

//TODO: HWDRC feature only exists on a few SKU currently, should list all microcode here
var microcodes = []string {
	"0x8d720210",
	"0x8d0002b0",
}

// labelNode label a node to indicate if it supports RDT/HWDRC
func updateNodeLabel(k8sCli *k8sclient.Clientset, rdt, hwdrc string) {
	node, err := k8sCli.CoreV1().Nodes().Get(context.TODO(), nodeName, meta_v1.GetOptions{})
	if err != nil || node == nil {
		klog.Errorf("Failed to get node: %v", err)
		klog.Warning("please ensure environment variable NODE_NAME has been set!")
	}

	if rdt != "" {
		node.Labels[rdtLabel] = rdt
	}

	if hwdrc != "" {
		node.Labels[hwdrcLabel] = hwdrc
	}

	k8sCli.CoreV1().Nodes().Update(context.TODO(), node, meta_v1.UpdateOptions{})
}

// labelNode label a node to indicate if it supports RDT/HWDRC
func labelNode(k8sCli *k8sclient.Clientset) bool {
	support := true

	// label the node
	node, err := k8sCli.CoreV1().Nodes().Get(context.TODO(), nodeName, meta_v1.GetOptions{})
	if err != nil || node == nil {
		klog.Errorf("Failed to get node: %v", err)
		klog.Warning("please ensure environment variable NODE_NAME has been set!")
		return false
	}

	// check if resctrl is supported
	if _, err := os.Stat("/sys/fs/resctrl"); err != nil {
		node.Labels[rdtLabel] = "no"
		support = false
	} else {
		// check if resctrl fs has been mounted
		if _, err := os.Stat("/sys/fs/resctrl/schemata"); err != nil {
			node.Labels[rdtLabel] = "disabled"
			support = false
		} else {
			node.Labels[rdtLabel] = "enabled"
		}
	}

	// check if HWDRC is supported
	node.Labels[hwdrcLabel] = "no"
	if cpuinfo, err := ioutil.ReadFile("/proc/cpuinfo"); err == nil {
		if idx := strings.Index(string(cpuinfo), "microcode"); idx >= 0 {
			microcode := string(cpuinfo[idx:])
			idx = strings.Index(microcode, "0x")
			for _, mc := range microcodes {
				if strings.HasPrefix(string(microcode[idx:]), mc) {
					node.Labels[hwdrcLabel] = "disabled"
				}
			}
		}
	}

	k8sCli.CoreV1().Nodes().Update(context.TODO(), node, meta_v1.UpdateOptions{})

	return support
}
