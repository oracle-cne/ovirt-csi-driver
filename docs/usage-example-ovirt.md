This document describes how to use the ovirt-csi-driver with an [oVirt](https://www.ovirt.org/download/index.html) or [OLVM](https://docs.oracle.com/en/virtualization/oracle-linux-virtualization-manager/) on-premises cluster 
and configure it to create PVs in a storage domain.
This allows you to create Pods that mount storage volumes as specified by a Kubernetes StorageClass.

*The instructions and commands below serve as an example and skip the TLS verification. Adapt where needed.*

# Prerequisite
* Create a dedicated user in oVirt/OLVM by executing the following command on your engine VM:
```
ovirt-aaa-jdbc-tool user add k8s
ovirt-aaa-jdbc-tool user password-reset k8s --password-valid-to="9999-12-31 23:59:59Z"
ovirt-aaa-jdbc-tool user edit k8s --attribute=firstName="ovirt-csi-driver"
```

Log in to the admin UI > Administration > Users > Add > `k8s` > Click on the created `k8s` user > Permissions > Add system permissions:
- ClusterAdmin (can be limited on specific clusters if needed)
- DiskCreator
- DiskOperator

You need the following information:
- oVirt-engine/OLVM API url: https://hostname.domain/ovirt-engine/api
- user and password from the previous step. (full username:  `k8s@internal`)
- CA cert of the API url (optional but recommended for production). In this example we skip the TLS verification.

Test the connection from your k8s VM node(s)
```
curl -k -u k8s@internal:mypassword [https://hostname.domain/ovirt-engine/api](https://hostname.domain/ovirt-engine/api)
```

# Installation
## Create Namespace
```
cat <<EOF | tee ovirt-csi-namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: ovirt-csi
EOF

kubectl apply -f ovirt-csi-namespace.yaml
```

## Create Secret
First, do a base 64 encoding of the values to be provided in the secret
```
export OVIRT_URL=$(echo -n https://hostname.domain/ovirt-engine/api | base64)
export OVIRT_USERNAME=$(echo -n k8s@internal | base64)
export OVIRT_PASSWORD=$(echo -n mypassword | base64)
export OVIRT_INSECURE=$(echo -n true | base64)
```

Then, create the secret
```
cat <<EOF | tee ovirt-csi-secret.yaml
ovirt-csi-secret.yaml 
apiVersion: v1
data:
  ovirt_url: "$OVIRT_URL"
  ovirt_username: "$OVIRT_USERNAME"
  ovirt_password: "$OVIRT_PASSWORD"
  ovirt_ca_bundle: ""
  ovirt_insecure: "$OVIRT_INSECURE"
kind: Secret
metadata:
  name:  ovirt-csi-creds
  namespace: ovirt-csi
type: Opaque
EOF

kubectl apply -f ovirt-csi-secret.yaml
```

## Create a custom Helm values file
- Make sure to adapt the values file based upon the version you're installing. You can find the Helm charts in the [Oracle Cloud Native Environment Application Catalog](https://github.com/oracle-cne/catalog)
- We need to prefix the image repo url with `container-registry.oracle.com`.
- Adapt the `ovirt` block in case you imported the CA cert.

In this case we're installing version 4.20.0: https://github.com/oracle-cne/catalog/tree/main/charts/ovirt-csi-driver-4.20.0

```
cat <<EOF | tee ovirt-csi-values.yaml
csiController:
  ovirtController:
    image:
      repository: container-registry.oracle.com/olcne/ovirt-csi-driver
      tag: v4.20.0-1
  prepareOvirtConfig:
    image:
      repository: container-registry.oracle.com/olcne/ovirt-csi-driver
      tag: v4.20.0-1
  csiAttacher:
    image:
      repository: container-registry.oracle.com/olcne/csi-attacher
      tag: v4.10.0
  csiProvisioner:
    image:
      repository: container-registry.oracle.com/olcne/csi-provisioner
      tag: v6.0.0
  csiResizer:
    image:
      repository: container-registry.oracle.com/olcne/csi-resizer
      tag: v1.14.0
  livenessProbe:
    image:
      repository: container-registry.oracle.com/olcne/livenessprobe
      tag: v2.17.0
csiNode:
  ovirtNode:
    image:
      repository: container-registry.oracle.com/olcne/ovirt-csi-driver
      tag: v4.20.0-1
  prepareOvirtConfig:
    image:
      repository: container-registry.oracle.com/olcne/ovirt-csi-driver
      tag: v4.20.0-1
  csiDriverRegistrar:
    image:
      repository: container-registry.oracle.com/olcne/csi-node-driver-registrar
      tag: v2.15.0
  livenessProbe:
    image:
      repository: container-registry.oracle.com/olcne/livenessprobe
      tag: v2.17.0
ovirt:
  caProvided: false
  insecure: true
  secretName: ovirt-csi-creds
EOF
```

## Deploy Helm chart
```
git clone https://github.com/oracle-cne/catalog.git
helm install ovirt-csi catalog/charts/ovirt-csi-driver-4.20.0 --namespace ovirt-csi --values ovirt-csi-values.yaml
```

Verify the pods are Running:
```
kubectl -n ovirt-csi get pods
NAME                                           READY   STATUS    RESTARTS   AGE
ovirt-csi-controller-plugin-69d5d7bbb4-dbf8g   5/5     Running   0          128m
ovirt-csi-node-plugin-4g677                    3/3     Running   0          128m
ovirt-csi-node-plugin-6r876                    3/3     Running   0          128m
ovirt-csi-node-plugin-746zx                    3/3     Running   0          128m
```

## Create StorageClass
Create a storage domain (or use an existing one) in the oVirt/OLVM admin console.<br/>
In this example, the storage domain is called `k8s-data`. Modify the values of the StorageClass further to your preference.
```
cat <<EOF | tee ovirt-csi-storageclass.yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: ovirt-lvm
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
parameters:
  storageDomainName: "k8s_data"
  thinProvisioning: "false"
  fsType: "xfs"
allowVolumeExpansion: true
provisioner: csi.ovirt.org
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
EOF

kubectl apply -f ovirt-csi-storageclass.yaml
```

## Test
### Create PersistentVolumeClaim
```
cat <<EOF | tee test-pvc.yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  namespace: default
  name: test-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: "1Gi"
  storageClassName: ovirt-lvm
EOF
kubectl apply -f test-pvc.yaml
```

### Create Pod
```
cat <<EOF | tee test-pod.yaml
apiVersion: v1
kind: Pod
metadata:
  namespace: default
  name: test-pod
spec:
  containers:
    - name: test-container
      image: busybox
      command: ["sleep", "3600"]
      volumeMounts:
        - name: test-volume
          mountPath: /mnt/data
  volumes:
    - name: test-volume
      persistentVolumeClaim:
        claimName: test-pvc
EOF
kubectl apply -f test-pod.yaml
```

### Verify pod & pvc
```
kubectl get pv
NAME                                       CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM              STORAGECLASS   VOLUMEATTRIBUTESCLASS   REASON   AGE
pvc-5002e239-7c23-4c09-bc7f-25546126444a   1Gi        RWO            Delete           Bound    default/test-pvc   ovirt-lvm      <unset>                          171m
```

```
kubectl get pvc
NAME        STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   VOLUMEATTRIBUTESCLASS   AGE
test-pvc    Bound    pvc-5002e239-7c23-4c09-bc7f-25546126444a   1Gi        RWO            ovirt-lvm      <unset>                 171m
```

```
kubectl exec -it test-pod -- sh
echo "hello ovirt" > /mnt/data/test.txt
```

### Cleanup
```
kubectl delete -f test-pod.yaml
kubectl delete -f test-pvc.yaml
```
