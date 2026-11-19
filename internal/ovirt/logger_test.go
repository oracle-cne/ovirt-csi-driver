package ovirt

import (
	"bytes"
	"context"
	"flag"
	"os"
	"strings"
	"testing"

	"k8s.io/klog"
)

func TestClientLoggerLevels(t *testing.T) {
	flags := flag.NewFlagSet("client-logger", flag.ContinueOnError)
	klog.InitFlags(flags)
	for name, value := range map[string]string{
		"logtostderr": "false", "alsologtostderr": "false", "stderrthreshold": "FATAL", "v": "0",
	} {
		original := flags.Lookup(name).Value.String()
		if err := flags.Set(name, value); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := flags.Set(name, original); err != nil {
				t.Error(err)
			}
		})
	}

	var output bytes.Buffer
	klog.SetOutput(&output)
	t.Cleanup(func() {
		klog.Flush()
		klog.SetOutput(os.Stderr)
	})

	for _, verbosity := range []string{"0", "3", "4", "0"} {
		t.Run("verbosity="+verbosity, func(t *testing.T) {
			t.Log("Configure verbosity, emit each severity through a context logger, flush, and inspect output")
			if err := flags.Set("v", verbosity); err != nil {
				t.Fatal(err)
			}
			output.Reset()
			logger := (clientLogger{}).WithContext(context.Background())
			logger.Debugf("debug message %d", 42)
			logger.Infof("info message %d", 42)
			logger.Warningf("warning message %d", 42)
			logger.Errorf("error message %d", 42)
			klog.Flush()

			if debug := strings.Contains(output.String(), "debug message 42"); debug != (verbosity == "4") {
				t.Fatalf("debug visibility at verbosity %s: %s", verbosity, output.String())
			}
			for _, expected := range []struct {
				message  string
				severity string
			}{
				{"info message 42", "I"},
				{"warning message 42", "W"},
				{"error message 42", "E"},
			} {
				found := false
				for _, line := range strings.Split(output.String(), "\n") {
					if strings.Contains(line, expected.message) {
						found = true
						if !strings.HasPrefix(line, expected.severity) {
							t.Errorf("expected severity %s: %s", expected.severity, line)
						}
					}
				}
				if !found {
					t.Errorf("missing %q at verbosity %s: %s", expected.message, verbosity, output.String())
				}
			}
		})
	}
}
