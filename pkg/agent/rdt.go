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
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"k8s.io/klog"
	"sigs.k8s.io/yaml"

)

const resctrlRoot = "/sys/fs/resctrl/"
const resctrlSchemataFile = "schemata"
const closidFile = "closid"
const cgroupCpusetRoot = "/sys/fs/cgroup/cpuset/"
const tasksFile = "tasks"

// cleanResctrlGroup removes resctrl group that not in 'groups'
func cleanResctrlGroup(groups []string) {
	if fis, err := ioutil.ReadDir(resctrlRoot); err == nil {
		for _, fi := range fis {
			if fi.IsDir() {
				found := false
				for _, group := range groups {
					if group == fi.Name() {
						found = true
						break
					}
				}

				if found {
					continue
				}

				path := filepath.Join(resctrlRoot, fi.Name(), resctrlSchemataFile)
				_, err := os.Lstat(path)
				if err == nil || os.IsExist(err) {
					os.Remove(filepath.Join(resctrlRoot, fi.Name()))
					klog.Info(filepath.Join(resctrlRoot, fi.Name()) + " is removed")
				}
			}
		}
    }
}

func updateResctrlGroup(dir, data string) {
	// create resctrl group if it doesn't exist
	if _, err := os.Lstat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			klog.Errorf("Failed to create directory %v: %v", dir, err)
			return
		}
	}

	if err := ioutil.WriteFile(filepath.Join(dir, resctrlSchemataFile), []byte(data+"\n"), 0600); err != nil {
		klog.Errorf("Failed to write %v to %v: %v", data, dir, err)
	}
}

func applyConfig(data *configData) {
	var rdtGroups, closids []string

	if data == nil {
		cleanResctrlGroup(rdtGroups)
		return
	}

	// parse configuration data
	for _, val := range *data {
		conf := make(map[string]interface{})
		if err := yaml.Unmarshal([]byte(val), &conf); err != nil {
			klog.Errorf("Failed to unmarshal configuration data: %v", err)
			return
		}

		for key, val := range conf {
			switch {
			case key == "rdt":
				groups := val.(interface{}).(map[string]interface{})
				for grp, rdtconf := range groups {
					rc := rdtconf.(map[string]interface{})
					for _, v := range rc {
						rcdata := v.(interface{}).(string)
						updateResctrlGroup(filepath.Join(resctrlRoot, grp), rcdata)
					}
					rdtGroups = append(rdtGroups, grp)
				}
			case key == "drc":
				dc := val.(interface{}).(map[string]interface{})
				groups := dc["low"].(interface{}).([]interface{})
				
				for _, grp := range groups {
					// Read closid, please ensure relative kernel patches have
					// been applied, otherwise there isn't "closid" file
					path := filepath.Join(resctrlRoot, grp.(interface{}).(string), closidFile)
					closid, err := ioutil.ReadFile(path)
					if err != nil {
						klog.Errorf("Failed to read closid (%q): %v", path, err)
						klog.Warning("please ensure hwdrc relative kernel patches have been applied")
					}

					closids = append(closids, string(closid))
				}

				enable := dc["enable"].(interface{}).(bool)
				if enable {
					getHwdrc().enable()
				} else {
					getHwdrc().disable()
				}
			}
		}
	}

	cleanResctrlGroup(rdtGroups)
	getHwdrc().setLowPriorityCloses(closids)
}

// readPids reads pids from a cgroup's tasks file
func readPids(tasksFile string) ([]string, error) {
	var pids []string

	f, err := os.OpenFile(tasksFile, os.O_RDONLY, 0644)
	if err != nil {
		klog.Errorf("Failed to open %q: %v", tasksFile, err)
		return nil, fmt.Errorf("Failed to open %q: %v", tasksFile, err)
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		pids = append(pids, s.Text())
	}
	if s.Err() != nil {
		klog.Errorf("Failed to read %q: %v", tasksFile, err)
		return nil, fmt.Errorf("Failed to read %q: %v", tasksFile, err)
	}

	return pids, nil
}

// writePids writes pids to a restctrl tasks file
func writePids(tasksFile string, pids []string) {
	f, err := os.OpenFile(tasksFile, os.O_WRONLY, 0644)
	if err != nil {
		klog.Errorf("Failed to write pids to %q: %v", tasksFile, err)
		return
	}
	defer f.Close()

	for _, pid := range pids {
		if _, err := f.Write([]byte(pid)); err != nil {
			if !errors.Is(err, syscall.ESRCH) {
				klog.Errorf("Failed to write pid %s to %q: %v", pid, tasksFile, err)
				return
			}
		}
	}
}

func assignRDTControlGroup(dir, rcgroup string) {
	if fis, err := ioutil.ReadDir(dir); err == nil {
		path := filepath.Join(dir, tasksFile)
		if _, err := os.Lstat(path); err == nil || os.IsExist(err) {
			klog.Infof("assignRDTControlGroup: %s, %s", path, rcgroup)
			if pids, err := readPids(path); err == nil {
				writePids(filepath.Join(resctrlRoot, rcgroup, tasksFile), pids)
			}
		}

		for _, fi := range fis {
			if fi.IsDir() {
				path := filepath.Join(dir, fi.Name())
				assignRDTControlGroup(path, rcgroup)
			}
		}
    }
}

func findPodAndAssign(dir, uid, rcgroup string) {
//	klog.Infof("findPodAndAssign: %s, %s, %s", dir, uid, rcgroup)
	if fis, err := ioutil.ReadDir(dir); err == nil {
		for _, fi := range fis {
			if fi.IsDir() {
				path := filepath.Join(dir, fi.Name())

				if strings.Contains(fi.Name(), uid) {
					assignRDTControlGroup(path, rcgroup)
					continue
				}

				findPodAndAssign(path, uid, rcgroup)
			}
		}
    }
}

// assignControlGroup adds the tasks of a pod into a resctrl control group
func assignControlGroup(uid, rcgroup string) {
	//the newset containerd has changed cgroup path delimiter from "_" to "-", wo we try both
	id := strings.Replace(uid, "-", "_", -1)
	findPodAndAssign(cgroupCpusetRoot, id, rcgroup)

	findPodAndAssign(cgroupCpusetRoot, uid, rcgroup)
}
