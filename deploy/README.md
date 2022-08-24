# Deployment

This CSI driver is deployed to an OpenShift cluster using the [ovirt-csi-driver-operator](https://github.com/openshift/ovirt-csi-driver-operator), which itself gets deployed by the [cluster-storage-operator](https://github.com/openshift/cluster-storage-operator). Examples are provided in the [deploy directory](./openshift/). 

## Deploying to plain kubernetes

In the [deploy/k8s directory](./k8s/), the necessary files are provided to deploy the ovirt-csi-driver to a plain kubernetes cluster hosted on oVirt. 