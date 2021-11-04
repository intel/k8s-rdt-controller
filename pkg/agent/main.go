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
	"flag"
	"io/ioutil"
	"math"
	"os"
	"strings"
	"time"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

var direct bool
var clientset *kubernetes.Clientset

func Main() {
	var config *rest.Config
	var err error

	kubeconfig := flag.String("kubeconfig", "", "absolute path to the kubeconfig file (optional, if run in cluster)")
	server := flag.String("server", "", "the relay server address")
	caFile := flag.String("ca-file", "", "absolute path to the root certificate file")
	certFile := flag.String("cert-file", "", "absolute path to the certificate file")
	keyFile := flag.String("key-file", "", "absolute path to the private key file")
	serverName := flag.String("cn", "", "the common name (CN) of the certificate")
	flag.BoolVar(&direct, "direct", false, "direct mode, default false. if true, it doesn't depends on a server to relay pod information")

	flag.Parse()

	if !direct {
		if *server == "" || *caFile == "" || *certFile == "" || *keyFile == "" || *serverName == "" {
			klog.Error("Arguments: server / ca-file / cert-file / key-file and cn should be set")
			return
		}
	}

	if *kubeconfig == "" {
		klog.Info("using in-cluster config")
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	}

	if err != nil {
		klog.Errorf("Failed to build config: %v", err)
		return
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Errorf("Failed creates clientset: %v", err)
		return
	}

	klog.Info("Node name: " + nodeName)
	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, meta_v1.GetOptions{})
	if err != nil || node == nil {
		klog.Errorf("Failed to get node: %v", err)
		klog.Warning("please ensure environment variable NODE_NAME has been set!")
		return
	}

	if !labelNode(clientset) {
		klog.Info("Seems this node doesn't support RDT or DRC. Please ensure resctrl fs is mounted")
		time.Sleep(math.MaxInt64)
	}

	getWatcher(clientset).start()

	if direct {
		for {
			time.Sleep(10 * time.Second)
		}
	} else {
		for {
			if err := startClient(*server, *caFile, *certFile, *keyFile, *serverName); err != nil {
				klog.Errorf("Client error: %v", err)
			}
		}
	}
}

func init() {
	// Node name is expected to be set in environment variable "NODE_NAME"
	nodeName = os.Getenv("NODE_NAME")

	if nodeName == "" {
		if hostname, err := ioutil.ReadFile("/etc/hostname"); err == nil {
			nodeName = strings.ToLower(strings.TrimSpace(string(hostname)))
		}
	}
}
