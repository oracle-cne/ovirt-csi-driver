package ovirt

import (
	"context"

	ovirtclientlog "github.com/ovirt/go-ovirt-client-log/v3"
	"k8s.io/klog"
)

// clientLogger preserves oVirt message severity and uses the driver's klog flags.
// Debug messages follow the driver's existing verbosity-4 convention.
type clientLogger struct{}

var _ ovirtclientlog.Logger = clientLogger{}

func (clientLogger) Debugf(format string, args ...interface{}) {
	klog.V(4).Infof(format, args...)
}

func (clientLogger) Infof(format string, args ...interface{}) {
	klog.Infof(format, args...)
}

func (clientLogger) Warningf(format string, args ...interface{}) {
	klog.Warningf(format, args...)
}

func (clientLogger) Errorf(format string, args ...interface{}) {
	klog.Errorf(format, args...)
}

// WithContext retains the adapter's context-independent logging behavior.
func (l clientLogger) WithContext(_ context.Context) ovirtclientlog.Logger {
	return l
}
