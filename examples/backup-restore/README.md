### How to Backup
* Deploy velero on the cluster using `velero-values.yaml`
* Update all the required details
    * bukect name
    * prefix
    * region
    * aws role arn if using kube2iam
    * You can also change the backup schedule ant ttl
* deploy it on the cluster

## TO-DO change helm to other option to the utility
```cassandraql
helm install --name velero --values velero-values.yaml --namespace velero --version 2.3.1 --wait
```

* Some commands to take backup other than scheduled once
    * backup for specific labels (you can pass multiple key values comma(,) )sperarted 
        ```cassandraql
      velero create backup <name>-backup --selector key=value --wait
        ```
    
    * backup for specific namespace
        ```cassandraql
      velero backup create nginx-backup --include-namespaces nginx --wait
        ```
      
    * backup specific resource/s
        ```cassandraql
        veleoro backup create secrets-backup --include-resources=secrets,serviceaccount,clusterrole,clusterrolebinding
        ```

### How to check backups
* Run below command
```cassandraql
velero backup get
```

### How to Restore
* Update the `velero-values.yaml` `restoreOnlyMode` to `true`
* Deploy to cluster where to restore
* Once it is deployed successfully check the backup from old cluster is available. Refer `How to check backups` section
* Restore commands:
    * Restore resources with specific label
        ```cassandraql
        velero restore create --from-backup <backup_name> --selector key=value
        ``` 
    * Restore resources with in a namespaces
        ```cassandraql
        velero restore create --from-backup <backup_name> --include-namespaces nginx
        ```
    * Restore Specific resources only
        ```cassandraql
        velero restore create --from-backup <backup_name> --include-resources=secrets,serviceaccount,clusterrole,clusterrolebinding
        ```