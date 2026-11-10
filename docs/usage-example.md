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
```shell
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
```shell
kubectl apply -f ./storage-class.yaml
```

# Step 3 - Create a PVC
Next create a PVC YAML file that uses the StorageClass.  Once you apply this file, Kubernetes will use
the ovirt-csi-driver to create the PV.  

Here is an example:  
```shell
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
```shell
kubectl apply -f ./pvc.yaml
```

Run the following command to see the pv that got created:
```shell
kubectl get pv
```

# Step 4 - Create a Pod to use the PVC
The following example shows a sample pod that mounts the volume by using a PVC.  

```shell
cat <<'EOF' > ./pod.yaml 
apiVersion: v1 
kind: Pod 
metadata:
  name: test
spec:
  containers:
  - image: container-registry.oracle.com/os/oraclelinux:8
    name: testpod
    command: ["sh", "-c", "while true; do ls -la /opt; echo this file system was made available using ovirt-csi-driver; sleep 1m; done"]
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
```shell
kubectl apply -f ./pod.yaml
```

Once the pod is running you can see the attached volume
```shell
kubectl exec -it test bash -- sh -c 'ls /demo'
```
# Step 5 - Cleanup
You can now delete the Pod, PVC, and PV that you created.  You can leave the StorageClass for further usage or delete it.  
Do not delete the CsiDriver.
```shell
kubectl delete -f ./pod.yaml
kubectl delete -f ./pvc.yaml
```
Once the PVC is deleted, then the PV will automatically get deleted, and the disk 
in your OLVM storage domain will likewise get deleted.


# Addendum 1 - Installing the ovirt-csi-driver from the catalog
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

# Addendum 2 - Using OLVM Disk Profiles
The ovirt-csi-driver allows you to specify an OLVM disk profile name in the StorageClass instead of the storage domain name.
The driver will query the OLVM server and get all the storage domains associated with the specified disk profile, 
then select the one that has the most space (least used).  You can use either diskProfileName or storageDomainName in a storageClass,
but not both at the same time.

For Example:
```text
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: oblock
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
provisioner: csi.ovirt.org
parameters:
  diskProfileName: test
  thinProvisioning: "true"
  fsType: ext4
```

# Summary
This document shows how to use the ovirt-csi-driver that was automatically installed OLVM CAPI cluster was created.
In addition, the document also describes the steps required for manually installing the ovirt-csi-driver from 
the catalog.


# Token renewal

The CSI Identity `Probe` checks the access token's JWT `exp` claim before testing
API access. The SDK reads this Unix-seconds deadline from the JWT payload when
each token is acquired. It does not use the outer SSO response's `exp` field,
which can report `9223372036854775807` even when the JWT has a finite lifetime.
JWT decoding is used only for renewal scheduling; the Engine validates API
authorization. No additional network request is needed to obtain the deadline.

By default, renewal becomes due at 90 percent of the token's lifetime, calculated from its
JWT `iat` (issued-at) and `exp` claims. If `iat` is absent, the SDK uses 90 percent
of the remaining lifetime measured when the token is acquired. A JWT valid for
four hours and forty-eight minutes is therefore due for renewal after four
hours, nineteen minutes and twelve seconds. The first probe
after the calculated threshold calls `ReconnectIfDue`, which uses the same
reconnect implementation as an explicit `Reconnect` call. Regular probes must continue running for proactive renewal; the
driver does not start a separate renewal timer. Probe frequency must be shorter
than the renewal lead time to ensure a check occurs before expiration.

Choose one of these arguments on the driver container:

```yaml
# Percentage of actual JWT validity; this is also the default.
args:
  - --token-renewal-percentage=90
```

```yaml
# Alternatively, renew five minutes after each successful token acquisition.
args:
  - --token-renewal-interval=5m
```

Percentage values must be integers from 1 through 99. The interval accepts
positive Go durations, such as `30s`, `5m`, or `1h30m`. Supplying both arguments,
including an explicitly supplied default percentage, rejects startup before any
API connection is created. Omitting both retains the 90-percent default.
The configured policy is logged at startup; deadline checks are logged at
verbosity 4. The arguments apply to controller and node driver processes.

Interval mode measures elapsed time from receipt of the token, independent of
JWT `iat` and `exp`, and also supports opaque tokens. It does not shorten the
interval to match token or SSO expiry. Set the interval, plus the maximum time
between probes, below the shortest applicable token/session lifetime.
Percentage mode follows each token's actual expiry, including session caps,
and therefore adapts when the configured lifetime changes. Neither mode starts
a background timer: reconnect occurs on the first probe at or after its deadline.

Both entry points serialize through the existing client reconnect mutex. The
scheduled entry point checks the deadline after acquiring that mutex, so a
concurrent reconnect can satisfy the deadline without a duplicate login.
Reconnect builds and authenticates a detached candidate, verifies API access
(or runs the configured verifier against that candidate), and then publishes
the authentication state to the stable SDK connection.

Successful reconnect updates the shared token, so existing controller, node,
context, SDK, and service handles use it on subsequent requests. Failed login,
verification, or publication retains the old
token and its deadline, allowing the next probe to retry. In percentage mode, a replacement must
contain a usable JWT expiry later than both the current time and the previous
token's expiry before a scheduled reconnect becomes active. Interval mode and explicit reconnect
also support opaque tokens and replacements with an unchanged expiry. Explicit reconnect does
not require the old token to be near expiry. Readiness
continues to reflect whether the API connection test succeeds with the active
token. Renewal attempts and results are logged without token or password values;
expiry checks are available at verbosity 4. In percentage mode, tokens without a usable JWT `exp`
claim produce a scheduling warning instead of falling back to an assumed
lifetime. Such tokens remain available for the API connection test.

With Keycloak 15, the JWT expiry is also capped at the SSO session maximum
(and applicable client session maximum). The percentage check therefore follows
the earlier deadline when a session limit is shorter than the configured access
token lifetime. Renewal uses a fresh password login with the configured
credentials, rather than the OAuth refresh-token grant: each successful login
creates a new Keycloak session with its own maximum lifetime. A maximum on the
original session does not carry over to the replacement session. No additional
SSO-session lifetime is assumed or configured in the driver.

Cached-token reads and deadline checks use the SDK's shared read lock. Initial
authentication and publication use its write lock. The active connection's
write lock is not held while a reconnect candidate authenticates or verifies,
so concurrent API calls can continue using the existing token. The SDK
connection, its transport, and its reader lock are never overwritten during a
reconnect. Initial construction stays lazy when verification is disabled.

The SDK calculates the configured acquisition-interval or JWT-percentage
renewal threshold and the driver checks it through `ReconnectIfDue`. Renewal
support extends the oVirt SDK and client. The `go.mod` replacements point to
`github.com/oracle-cne/go-ovirt` and `github.com/oracle-cne/go-ovirt-client/v2`.
Both replacements pin published revisions from the forks' `1-token-refresh`
branches. `GOWORK=off go mod vendor` regenerates the snapshot from those
revisions without sibling checkouts. See `hack/forks/README.md` for exact pins,
transferable patches, and published dependency build commands.

The authoritative renewal code and tests are in the sibling SDK and v2 client
fork checkouts, based on the exact dependency commits listed in
`hack/forks/README.md`. Regenerated vendor snapshots do not include dependency
test files. From the parent development directory, run the renewal tests with:

```shell
export GOWORK="$PWD/go.work.local"
go test -race ./go-ovirt
go test -race ./go-ovirt-client -run 'Test(Reconnect|MockScheduledReconnect)' -client=mock
go test -race ./ovirt-csi-driver/pkg/service
```
