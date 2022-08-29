module github.com/ovirt/csi-driver

go 1.16

require (
	github.com/container-storage-interface/spec v1.2.0
	github.com/golang/protobuf v1.5.2
	github.com/kubernetes-csi/csi-lib-utils v0.7.0
	github.com/ovirt/go-ovirt-client v1.0.0
	github.com/ovirt/go-ovirt-client-log-klog/v2 v2.0.0
	github.com/ovirt/go-ovirt-client-log/v3 v3.0.0
	github.com/pkg/errors v0.9.1
	golang.org/x/net v0.0.0-20210428140749-89ef3d95e781
	golang.org/x/sys v0.0.0-20210603081109-ebe580a85c40
	google.golang.org/grpc v1.29.1
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20210527160623-6fdb442a123b
	sigs.k8s.io/controller-runtime v0.9.2
)

replace github.com/ovirt/go-ovirt-client v0.6.1-0.20210927190907-b4e32fab2754 => github.com/Gal-Zaidman/go-ovirt-client v0.0.0-20211012125710-ab43b89c03e1
