package init_cluster

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/greenplum-db/gpdb/gp/test/integration/testutils"
)

var (
	hostList []string
	hostfile = flag.String("hostfile", "", "file containing list of hosts")
	coordinatorDatadir = testutils.DEFAULT_COORDINATOR_DATADIR
)

func TestMain(m *testing.M) {
	flag.Parse()

	// Hostfile is required to distinguish between single host and multi host testing
	// If no file is provided, run only single host tests by creating the file
	if *hostfile == "" {
		file, err := os.CreateTemp("", "")
		if err != nil {
			fmt.Printf("could not create hostfile: %v, and no hostfile provided", err)
			os.Exit(1)
		}

		*hostfile = file.Name()
		err = os.WriteFile(*hostfile, []byte("localhost"), 0777)
		if err != nil {
			fmt.Printf("could not create hostfile: %v, and no hostfile provided", err)
			os.Exit(1)
		}

		coordinatorDatadir = "/tmp/demo/-1"
	}

	hostList = testutils.GetHostListFromFile(*hostfile)
	if len(hostList) == 0 {
		fmt.Printf("no hosts provided in the hostfile %q", *hostfile)
		os.Exit(1)
	}

	err := testutils.ConfigureAndStartServices(*hostfile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	os.Setenv("COORDINATOR_DATA_DIRECTORY", coordinatorDatadir)
	os.Setenv("PGPORT", strconv.Itoa(testutils.DEFAULT_COORDINATOR_PORT))

	exitCode := m.Run()
	testutils.RunStop("services") //nolint

	os.Exit(exitCode)
}
