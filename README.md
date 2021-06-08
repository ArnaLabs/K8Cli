Reach out to us @www.arna.cloud for any questions

# K8Cli
Multi Cloud K8s CLuster Setup


# Enterprise Version Support:
* Helm Backup and Restore       -- Completed
* SSO LDAP (Dex) Authentication -- Evaluatiom
* Dynamics number of subnets    -- Evaluation


# Supported in Open Source 
* Cluster Creation              -- Completed on AWS
* Canary Upgrades               -- Evaluatiom
* AWS, AKS and GKE Support      -- AWS Completed
* Encryption at Rest            -- Completed
* Addons Deployment with helm   -- Completed
* Support 3 Private and Public subnets max - Review

##Commands:
### Init Cluster Management
```
./K8Cli --operation init --context test-eks9
 you can give kubeconfig path here 
```

### Setup EKS Cluster
```
./K8Cli --operation cluster --context test-eks9
Run the Cli in the path where you have your folder K8CLI/<cluster-name>
```
### get cluster config on local
it will create kubeconfig under ~/.kube/config or if there is existing file it will do a safe merge with existing contexts
```
aws eks update-kubeconfig --name <cluster_name> --alias <alias_name>
```

### Setup Add-Ons
```

./K8Cli --operation addons --context test-eks9
``` 

### Setup namespace
```
./K8Cli --operation namespace --context test-eks9
```

### Setup Resource Quota
```
./K8Cli --operation resourcequota --context test-eks9
```

#### Examples

#### TODO
1. restrict control plane with cidr  -- Review
1. Lable nodes                       -- Review
2. Namespace Quota                   -- Completed
   * resource limits
   * pods 50
   * storage 50Gi
3. a chart                           -- Completed
    * create namespace
    * user with admin access
    * user with readonly access
4. k8s netwok policies               -- Evaluation
5. Compare CFT checksum before apply -- Evaluation
    
### 17/02/2021
* version command                       -- Pending
* backup take option with config file   -- Review 
* subnets support 4, 6, 8 with fixed CFT Samples -- Review
* eks cluster creation yaml doc         -- Pending

## 03/04/2021
* add namespace management in the operations -- Completed
* documentations                             -- Pending

## 23/02/2021
* Azure Support                              -- Review


// Udated commands:
