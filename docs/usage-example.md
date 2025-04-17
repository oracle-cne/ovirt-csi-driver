This document describes how to install the ovirt-csi-driver into an OLVM CAPI cluster 
and configure it to create PVs in an OLVM storage domain.  

# Step 1: Create OVLM CAPI cluster  
You wil need an OLVM CAPI cluster with a configured storage domain.  See [Creating an OVLM CAPI cluster](https://github.com/oracle-cne/ocne/blob/main/doc/cluster-management/olvm.md)  
When you create an OLVM Kubernetes cluster with the `ocne cluster start --provider olvm`, 
then the `ocne` client automatically creates the secret, configmap, then installs the ovirt-csi-driver.

# Step 2: Create a StorageClass 
Before using the ovirt-csi-driver, you need to create a StorageClass, specifying the following:

* provisioner - this must be set to csi.ovirt.og  
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
Create a PVC YAML file that uses the storage class.  For example
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
The following example shows a sample pod that mounts the PVC.
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

# Summary
This document shows how to use the ovirt-csi-driver that was automatically installed OLVM CAPI cluster was created.

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

Now run the following command to generate the secret YAML file.
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
Apply the secret
```
kubectl --kubeconfig $KUBEOLVM apply -f ./ovirt-csi-secret.yaml 
```

## Step 2: Create the configmap used by the ovirt-csi-driver
This is only needed if your CA cert is not a well-known cert in the trust store.
Get the existing ca.crt configmap used by the OLVM CAPI controller.
By default, the configmap is in namespace `olvm-cluster`, configmap name `<cluster-name>-ovirt-ca`.
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
  
The new configmap that you need to create has the same contents by different namespace/name.
Let's call this file `./ovirt-csi-configmap.yaml`.
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
  
Apply the configmap
```
kubectl --kubeconfig $KUBEOLVM apply -f ./ovirt-csi-configmap.yaml
````

## Step 3: Install the ovirt-csi-driver from the catalog
Using the `ocne` CLI, install the `ovirt-csi-driver` into the OLVM CAPI cluster using the following command:
```text
export $KUBEOVLM <olvm kubeconfig file>
ocne application install --catalog embedded --name ovirt-csi-driver --kubeconfig $KUBEOVLM
```

You can provide an override file to change any of the values defined in the values.yaml file of the catalog application.