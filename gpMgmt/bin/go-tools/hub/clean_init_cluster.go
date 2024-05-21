package hub

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/greenplum-db/gp-common-go-libs/gplog"
	"github.com/greenplum-db/gpdb/gp/constants"
	"github.com/greenplum-db/gpdb/gp/idl"
	"github.com/greenplum-db/gpdb/gp/utils"
)

/*
function to convert entries in entries.txt file into hashmap
*/
func CreateHostDataDirMap(entries []string, file string) (map[string][]string, error) {

	hostDataDirMap := make(map[string][]string)

	for _, entry := range entries {
		fields := strings.Fields(entry)
		if len(fields) == 2 {
			host := fields[0]
			dataDir := fields[1]
			hostDataDirMap[host] = append(hostDataDirMap[host], dataDir)
		} else {
			return nil, errors.New("invalid entries in map")
		}
	}
	return hostDataDirMap, nil
}

/*
rpc to cleanup the data directories in case gp init cluster fails.
*/
func (s *Server) CleanInitCluster(context.Context, *idl.CleanInitClusterRequest) (*idl.CleanInitClusterReply, error) {

	var err error
	fileName := filepath.Join(s.LogDir, constants.CleanFileName)

	_, err = utils.System.Stat(fileName)
	if err != nil {
		gplog.Info("Cluster is clean")
		return &idl.CleanInitClusterReply{}, nil
	}

	//Read entries from fileName
	entries, err := utils.ReadEntriesFromFile(fileName)

	if err != nil {
		return &idl.CleanInitClusterReply{}, utils.LogAndReturnError(fmt.Errorf("init clean cluster failed err: %v", err))
	}
	// Create hostDataDirMap from entries
	hostDataDirMap, err := CreateHostDataDirMap(entries, fileName)
	if err != nil {
		return &idl.CleanInitClusterReply{}, utils.LogAndReturnError(fmt.Errorf("invalid entries in cleanup file"))
	}

	request := func(conn *Connection) error {
		var wg sync.WaitGroup

		hostname := hostDataDirMap[conn.Hostname]
		errs := make(chan error, len(hostname))

		for _, dir := range hostname {
			dir := dir
			wg.Add(1)

			go func(dirname string) {
				defer wg.Done()

				gplog.Debug("Removing Data Directories: %s", dir)
				_, err := conn.AgentClient.RemoveDirectory(context.Background(), &idl.RemoveDirectoryRequest{
					DataDirectory: dir,
				})
				if err != nil {
					errs <- utils.FormatGrpcError(err)
				}
			}(dir)
		}

		wg.Wait()
		close(errs)

		var err error
		for e := range errs {
			err = errors.Join(err, e)
		}
		return err
	}

	defer os.Remove(fileName)

	return &idl.CleanInitClusterReply{}, ExecuteRPC(s.Conns, request)
}
