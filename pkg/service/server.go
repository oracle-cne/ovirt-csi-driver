package service

import (
	"context"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
	"google.golang.org/grpc"
	"k8s.io/klog"
)

// NonBlockingGRPCServer defines Non blocking GRPC server interfaces
type NonBlockingGRPCServer interface {
	// Start services at the endpoint
	Start(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer)
	// Wait for the service to stop
	Wait()
	// Stop stops the service gracefully
	Stop()
	// ForceStop stops the service forcefully
	ForceStop()
}

func NewNonBlockingGRPCServer() NonBlockingGRPCServer {
	return &nonBlockingGRPCServer{}
}

// NonBlocking server
type nonBlockingGRPCServer struct {
	wg     sync.WaitGroup
	server *grpc.Server
}

func (s *nonBlockingGRPCServer) Start(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer) {

	s.wg.Add(1)

	go s.serve(endpoint, ids, cs, ns)

	return
}

func (s *nonBlockingGRPCServer) Wait() {
	s.wg.Wait()
}

func (s *nonBlockingGRPCServer) Stop() {
	s.server.GracefulStop()
}

func (s *nonBlockingGRPCServer) ForceStop() {
	s.server.Stop()
}

func (s *nonBlockingGRPCServer) serve(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer) {
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(logGRPC),
	}

	u, err := url.Parse(endpoint)

	if err != nil {
		klog.Fatal(err.Error())
	}

	var addr string
	if u.Scheme == "unix" {
		addr = u.Path
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			klog.Fatalf("Failed to remove %s, error: %s", addr, err.Error())
		}

		listenDir := filepath.Dir(addr)
		if _, err := os.Stat(listenDir); err != nil {
			if os.IsNotExist(err) {
				klog.Fatalf("Expected Kubelet plugin watcher to create parent dir %s but did not find such a dir", listenDir)
			} else {
				klog.Fatalf("Failed to stat %s, error: %s", listenDir, err.Error())
			}
		}

	} else if u.Scheme == "tcp" {
		addr = u.Host
	} else {
		klog.Fatalf("%v endpoint scheme not supported", u.Scheme)
	}

	klog.V(4).Infof("Start listening with scheme %v, addr %v", u.Scheme, addr)
	listener, err := net.Listen(u.Scheme, addr)
	if err != nil {
		klog.Fatalf("Failed to listen: %v", err)
	}

	server := grpc.NewServer(opts...)
	s.server = server

	if ids != nil {
		klog.Infof("Registering Identity Server")
		csi.RegisterIdentityServer(server, ids)
	}
	if cs != nil {
		klog.Infof("Registering Controller Server")
		csi.RegisterControllerServer(server, cs)
	}
	if ns != nil {
		klog.Infof("Registering Node Server")
		csi.RegisterNodeServer(server, ns)
	}

	klog.V(4).Infof("Listening for connections on address: %#v", listener.Addr())

	if err := server.Serve(listener); err != nil {
		klog.Fatalf("Failed to serve: %v", err)
	}

}

func logGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	klog.V(4).Infof("%s called with request: %+v", info.FullMethod, protosanitizer.StripSecrets(req))
	resp, err := handler(ctx, req)
	if err != nil {
		klog.Errorf("%s returned with error: %v", info.FullMethod, err)
	} else {
		klog.V(4).Infof("%s returned with response: %+v", info.FullMethod, protosanitizer.StripSecrets(resp))
	}
	return resp, err
}
