package testutils

import (
	"fmt"
	"os"
)

const (
	DEFAULT_COORDINATOR_PORT    = 7000
	DEFAULT_COORDINATOR_DATADIR = "/data/qddir/gpseg-1"
)

var (
	GpHome                   = os.Getenv("GPHOME")
	DefaultHost              = "localhost"
	DefaultConfigurationFile = fmt.Sprintf("%s/gp.conf", GpHome)
	CertificateParams        = []string{
		"--ca-certificate", "/tmp/certificates/ca-cert.pem",
		"--ca-key", "/tmp/certificates/ca-key.pem",
		"--server-certificate", "/tmp/certificates/server-cert.pem",
		"--server-key", "/tmp/certificates/server-key.pem",
	}
	CommonHelpText  = []string{"Usage:", "Flags:", "Global Flags:"}
	DefaultHostfile = "/tmp/hostlist"
)
