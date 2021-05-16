### k8-cloud

This tools helps you to setup kubernetes cluster on multiple cloud proverers like AWS, Azure and also deploy addons like nginx-controller, prometheus, etc.


### How to install
* Download the package from https://github.com/K8-Cloud/k8-cloud/releases
* setup the PATH

#### Setup EKS Cluster
```
./k8-cloud -o cluster -c examples/eks-cluster.yml
```
#### Setup Add-Ons
```
./k8-cloud -o addons -c examples/addon.yaml --context  cluster-name
``` 