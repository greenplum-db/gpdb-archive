package status

import (
	"flag"
	"github.com/greenplum-db/gpdb/gp/test/integration/testutils"
	"github.com/greenplum-db/gpdb/gp/utils"
	"os"
	"testing"
)

var (
	p          = utils.GetPlatform()
	configCopy = "config_copy.conf"
	hostfile   = flag.String("hostfile", "", "file containing list of hosts")
)

func TestMain(m *testing.M) {
	flag.Parse()
	// if hostfile is not provided as input argument, create it with default host
	if *hostfile == "" {
		*hostfile = testutils.DefaultHostfile
		_ = os.WriteFile(*hostfile, []byte(testutils.DefaultHost), 0644)
	}
	exitVal := m.Run()
	tearDownTest()

	os.Exit(exitVal)
}

func tearDownTest() {
	testutils.CleanupFilesOnHub(configCopy)
	testutils.DisableandDeleteServiceFiles(p)
}
