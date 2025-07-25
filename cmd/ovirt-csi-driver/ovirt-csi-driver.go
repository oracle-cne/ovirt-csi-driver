package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ovirt/csi-driver/pkg/ovirt/ovclient"
	"math/rand"
	"os"
	"time"

	"github.com/ovirt/csi-driver/internal/ovirt"
	ovconfig "github.com/ovirt/csi-driver/pkg/config"
	"github.com/ovirt/csi-driver/pkg/service"

	ovirtclient "github.com/ovirt/go-ovirt-client/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	endpoint            = flag.String("endpoint", "unix:/csi/csi.sock", "CSI endpoint")
	namespace           = flag.String("namespace", "", "Namespace to run the controllers on")
	ovirtConfigFilePath = flag.String("ovirt-conf", "", "Path to ovirt api config")
	nodeName            = flag.String("node-name", "", "The node name - the node this pods runs on")
)

func init() {
	klog.InitFlags(flag.CommandLine)
}

func main() {
	flag.Set("logtostderr", "true")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	handle()
	os.Exit(0)
}

func handle() {

	// Validate server connection
	c, err := ovconfig.GetOvirtConfig()
	if err != nil {
		klog.Fatal(fmt.Errorf("failed to get ovirt config: %v", err))
	}
	ovclient.GetOVClient(c)
	if err != nil {
		klog.Fatal(fmt.Errorf("error creating ovclient: %v", err))
	}

	if service.VendorVersion == "" {
		klog.Fatalf("VendorVersion must be set at compile time")
	}
	klog.V(2).Infof("Driver vendor %v %v", service.VendorName, service.VendorVersion)

	ovirtClient, err := ovirt.NewClient()
	if err != nil {
		klog.Fatalf("Failed to initialize ovirt client %s", err)
	}
	klog.Infof("Success verifying connection to ovirt server")

	klog.Infof("Calling config.GetConfig()\n")

	// Get a config to talk to the apiserver
	restConfig, err := config.GetConfig()
	if err != nil {
		klog.Fatal(err)
	}

	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		klog.Infof("error getting rest config")
		klog.Fatal(err)
	}

	klog.Infof("Testing access to Kubernets API server\n")
	nodeList, err := clientSet.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		klog.Fatal(err)
	}
	if len(nodeList.Items) == 0 {
		klog.Fatal(fmt.Errorf("no nodes found in Kubernetes cluster"))
	}
	klog.Infof("Found %d nodes in cluster\n", len(nodeList.Items))

	opts := manager.Options{
		Namespace:          *namespace,
		MetricsBindAddress: "0",
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(restConfig, opts)
	if err != nil {
		klog.Fatal(err)
	}
	go func() {
		if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
			klog.Fatal(err)
		} else {
			klog.Info("Manager stopped gracefully.")
		}
	}()
	// get the node object by name and pass the VM ID because it is the node
	// id from the storage perspective. It will be used for attaching disks
	var nodeId ovirtclient.VMID

	if *nodeName != "" {
		get, err := clientSet.CoreV1().Nodes().Get(context.Background(), *nodeName, metav1.GetOptions{})
		if err != nil {
			klog.Fatal(err)
		}
		nodeId = ovirtclient.VMID(get.Status.NodeInfo.SystemUUID)
	}

	driver := service.NewOvirtCSIDriver(ovirtClient, nodeId)

	driver.Run(*endpoint)
}
