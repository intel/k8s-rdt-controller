# k8s-rdt-controller

k8s-rdt-controller is a helper for applying RDT / HWDRC in a kubernetes cluster.

-  It adds label(s) on nodes in the cluster to indicate if RDT / HWDRC feature is supported by the nodes.
-  It monitors RDT / HWDRC configuration and applies the configuration on relative nodes automatically.
-  It montiors the information of pods and puts the pods in the control of  RDT / HWDRC on demand.

## Label a node

k8s-rdt-controller checks the state of RDT / HWDRC feature of the cluster nodes. It adds "RDT='state'" and "HWDRC='state'" label on the nodes. 'state' can be 'no', 'disabled' or 'enabled'. User can ensure a pod is assigned to a node with RDT enabled by node selector. e.g.
```
  nodeSelector:
    RDT: enabled
```

Detail about node selector please refer to: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/

### Add a node into a node group

There may be different kinds of machines in a cluser or user would like apply different RDT / HWDRC configuration due to some reason. User can group the nodes by "ngroup" label. If a node has a "ngroup=grp1" label, the node is taken as belonging to node group "grp1".

## Monitor and apply RDT / HWDRC configuration

There are three types of configurations:

-  configuration for a node named NODE_NAME: rc-config.node.{NODE_NAME}
-  configuration for nodes in GROUP_NAME node group: rc-config.group.{GROUP_NAME}
-  default configuration: rc-config.default

User should create configuration for a node or a group of nodes by ConfigMap. The ConfigMap named rc-config.node.{NODE_NAME} provides the configuration for a node named NODE_NAME. If a node belongs to a node group and the node specific ConfigMap rc-config.node.{NODE_NAME} doesn't exist, the ConfigMap named rc-config.group.{GROUP_NAME} will be applied. If a node doesn't belongs to any node group and the node specific ConfigMap doesn't exist, the ConfigMap named rc-config.default will be applied. Please note when you create a node group, that's to say add label "ngroup=xxx" on some nodes, generally you should create ConfigMap rc-config.group.xxx for the nodes in the node group at the same time. Otherwise, you should create ConfigMap rc-config.node.{node_name} for each node in the node group.

Following is an example of RDT / HWDRC configuration, the CONFIG_NAME can be rc-config.node.{NODE_NAME}, rc-config.group.{GROUP_NAME} or rc-config.default. 

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ${CONFIG_NAME}
  namespace: rc-config 
data:
  rc.conf: |
    rdt:
      group1:
        llc: <schemata>
        mb: <schemata>
      group2:
        llc: <schemata>
        mb: <schemata>
      group3:
        llc: <schemata>
        mb: <schemata>
    drc:
      enable: true
      low: ["group1"]
```

Typically, a RDT control group configuration should be as following

```
GROUP_NAME:
  LABEL_1: <schemata>
  LABEL_2: <schemata>
  ...
  LABEL_n: <schemata>
```

The LABEL_{n} is illustrative. k8s-rdt-controller will create a {GROUP_NAME} directory in /sys/fs/resctrl and write the {schemata}s into /sys/fs/resctrl/{GROUP_NAME}/schemata one by one directly. {schemata}s should follow resctrl schemata syntax.

HWDRC configuration is very simple. It only contains two key-value items. The value of 'enable' key indicates if HWDRC should be enabled or disabled. Please note once HWDRC is enabled, the MB setting of all RDT groups will be invalided by HWDRC. The value of 'low' key is a list of RDT control group name. The tasks in these groups are considered as low priority. HWDRC will ensure the performance of the tasks aren't in the groups in a higher priority.

k8s-rdt-controller monitors the ConfigMaps in the kubernetes cluster and applies the configuration to relative node(s) once a ConfigMap is added or updated.

##  Monitor pod information and put the pod in the controll of RDT / HWDRC

User should specify the RDT control group of a pod by adding a "rcgroup" label for the pod. e.g. if a pod with label "rcgroup=group2", the pod will be added to "group2" RDT control group by k8s-rdt-controller automatically.

## How to build

local build

```
# make
```

Build in docker

```
# make docker
```

## How to run

  1. create configuration
```
# kubectl apply -f example-config.yaml
```

  2. run k8s-rdt-controller

```
# kubectl apply -f k8s-rdt-controller.yaml
```

  3. create your pod
```
# kubectl apply -f example-pod.yaml
```
