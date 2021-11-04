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
	"os/exec"
	"strings"

	"k8s.io/klog"
)

const disableScript = "./scripts/disable_hwdrc.sh"
const enableScript = "./scripts/enable_hwdrc.sh"

type hwdrc struct {
	// if HWDRC is enabled
	enabled bool

	// low priority CLOSes
	closes []string
}

var singleton_drc *hwdrc

func getHwdrc() *hwdrc {
	if singleton_drc == nil {
		singleton_drc = &hwdrc{}
	}

	return singleton_drc
}

func (h *hwdrc) disable() {
	if !h.enabled {
		return
	}

	cmd := exec.Command("sh", "-c", disableScript)

	if _, err := cmd.Output(); err != nil {
		klog.Errorf("Failed to execute script (%q): %v", disableScript, err)
		return
	}

	h.enabled = false
	updateNodeLabel(clientset, "", "disabled")
	klog.Info("HWDRC is disabled")
}

func (h *hwdrc) enable() {
	closesStr := strings.Join(h.closes, ",")

	cmd := exec.Command("sh", "-c", enableScript, closesStr)

	if _, err := cmd.Output(); err != nil {
		klog.Errorf("Failed to execute script (%q): %v", enableScript, err)
		return
	}

	h.enabled = true
	updateNodeLabel(clientset, "", "enabled")
	klog.Info("HWDRC is enabled")
}

func (h *hwdrc) setLowPriorityCloses(closes []string) {
	h.closes = make([]string, len(closes))
	copy(h.closes, closes)

	if h.enabled {
		h.enable()
	}
}
