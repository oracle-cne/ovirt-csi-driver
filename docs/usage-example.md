This document describes how to use the ovirt-csi-driver with an OLVM CAPI cluster 
and configure it to create PVs in an OLVM storage domain.  This allows you to
create Pods that mount OLVM storage volumes as specified by a Kubernetes StorageClass. 

# Step 1: Create OLVM CAPI cluster  
First, you need an OLVM CAPI cluster with a configured storage domain.  See [Creating an OVLM CAPI cluster](https://github.com/oracle-cne/ocne/blob/main/doc/cluster-management/olvm.md)  
When you create an OLVM Kubernetes cluster with the `ocne cluster start --provider olvm`, 
the `ocne` client automatically installs the ovirt-csi-driver, creating the required
secret, configmap, and CsiDriver.  There are no extra steps required at this point.

# Step 2: Create a StorageClass 
Before using the ovirt-csi-driver, you need to create a StorageClass that uses the CsiDriver installed previously.  
  
You need to specify the following:
* provisioner - this must be set to csi.ovirt.org, which is the CsiDriver name.
* storageDomainName - this is the name of your OLVM storage domain where PV volumes will be created.  This domain must exist.  
* thinProvisioning - set true or false  
* fsType - set file system type.  The driver will format the volume if there is no file system.  

Create the StorageClass yaml file.  
```
cat <<'EOF' > ./storage-class.yaml 
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: oblock
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
provisioner: csi.ovirt.org
parameters:
  # the name of the oVirt storage domain. "Default" is just an example.
  storageDomainName: oblock
  thinProvisioning: "true"
  fsType: ext4
EOF  
```

Apply the YAML file:
```
kubectl apply -f ./storage-class.yaml
```

# Step 3 - Create a PVC
Next create a PVC YAML file that uses the StorageClass.  Once you apply this file, Kubernetes will use
the ovirt-csi-driver to create the PV.  

Here is an example:  
```
cat <<'EOF' > ./pvc.yaml 
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: 1g-ovirt-disk
spec:
  storageClassName: oblock
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
EOF  
```
Apply the YAML file:
```
kubectl apply -f ./pvc.yaml
```

Run the following command to see the pv that got created:
```
kubectl get pv
```

# Step 4 - Create a Pod to use the PVC
The following example shows a sample pod that mounts the volume by using a PVC.  

```
cat <<'EOF' > ./pod.yaml 
apiVersion: v1 
kind: Pod 
metadata:
  name: test
spec:
  containers:
  - image: container-registry.oracle.com/os/oraclelinux:8
    name: testpod
    command: ["sh", "-c", "while true; do ls -la /opt; echo this file system was made availble using ovirt-csi-driver; sleep 1m; done"]
    imagePullPolicy: IfNotPresent
    volumeMounts:
    - name: pv0002
      mountPath: "/demo"
  volumes:
  - name: pv0002
    persistentVolumeClaim:
      claimName: 1g-ovirt-disk
EOF        
```

Apply the YAML file:
```
kubectl apply -f ./pod.yaml
```

Once the pod is running you can see the attached volume
```
kubectl exec -it test bash -- sh -c 'ls /demo'
```
# Step 5 - Cleanup
You can now delete the Pod, PVC, and PV that you created.  You can leave the StorageClass for further usage or delete it.  
Do not delete the CsiDriver.
```
kubectl delete -f ./pod.yaml
kubectl delete -f ./pvc.yaml
```
Once the PVC is deleted, then the PV will automatically get deleted, and the disk 
in your OLVM storage domain will likewise get deleted.


# Addendum - Installing the ovirt-csi-driver from the catalog
**NOTE** This is only needed if you wish to install the ovirt-csi-driver manually.
This automatic installation described above in step 1 is the preferred mechanism.  
With that said, the instructions to do these steps manually are described below.

## Step 1: Create the secret used by the ovirt-csi-driver
Do a base64 encoding all the fields specified in the YAML file below.

For example, base64 encode the url (be sure to use -n):
```
OVIRT_URL=echo -n https://my.example.com/ovirt-engine/api | base64
OVIRT_PASSWORD=echo -n <password> | base64
OVIRT_USERNAME=echo -n <username> | base64
```

Run the following command to generate the Secret YAML file:
```text
envsubst > ./ovirt-csi-secret.yaml 
apiVersion: v1
data:
  ovirt_url: $OVIRT_URL
  ovirt_password: $OVIRT_PASSWORD
  ovirt_username: $OVIRT_USERNAME
kind: Secret
metadata:
  name:  ovirt-csi-creds
  namespace: ovirt-csi
type: Opaque
```
Apply the Secret:
```
kubectl --kubeconfig $KUBEOLVM apply -f ./ovirt-csi-secret.yaml 
```

## Step 2: Create the ConfigMap used by the ovirt-csi-driver
This is not needed if your CA certificate is signed by a trusted CA and in the local trust store.  

To create the ConfigMap CA, get the existing ca.crt ConfigMap used by the OLVM CAPI controller.
By default, the ConfigMap is in namespace `olvm-cluster`, ConfigMap name `<cluster-name>-ovirt-ca`.
Dump the contents of the configmap in a file, change the namespace/name and kubectl apply the file.  

For example:
```text
 k get cm -n olvm-cluster  demo-ovirt-ca -o yaml
apiVersion: v1
data:
  ca.crt: |
    -----BEGIN CERTIFICATE-----
    MIIEUzCC...
    C1dfawYJCA==
    -----END CERTIFICATE-----
kind: ConfigMap
metadata:
  name: demo-ovirt-ca
  namespace: olvm-cluster
```
  
The new ConfigMap that you need to create has the same contents by different namespace/name.
Name this file `./ovirt-csi-configmap.yaml`:
```text
apiVersion: v1
data:
  ca.crt: |
    -----BEGIN CERTIFICATE-----
    MIIEUzCC...
    C1dfawYJCA==
    -----END CERTIFICATE-----
kind: ConfigMap
metadata:
  name: ovirt-csi-ca.crt 
  namespace: ovirt-csi
```
  
Apply the ConfigMap:
```
kubectl --kubeconfig $KUBEOLVM apply -f ./ovirt-csi-configmap.yaml
```

## Step 3: Install the ovirt-csi-driver from the catalog
Using the `ocne` CLI, install the `ovirt-csi-driver` into the OLVM CAPI cluster.  This will also install the CsiDriver.  

Run the following command:
```text
export $KUBEOVLM <olvm kubeconfig file>
ocne application install --catalog embedded --name ovirt-csi-driver --kubeconfig $KUBEOVLM
```

You can provide an override file to change any of the values defined in the values.yaml file of the catalog application.

# Summary
This document shows how to use the ovirt-csi-driver that was automatically installed OLVM CAPI cluster was created.
In addition, the document also describes the steps required for manually installing the ovirt-csi-driver from 
the catalog.
