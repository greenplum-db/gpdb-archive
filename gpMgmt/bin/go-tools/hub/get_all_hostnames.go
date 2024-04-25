package hub

import (
	"context"
	"fmt"
	"sync"

	"github.com/greenplum-db/gpdb/gp/utils"

	"github.com/greenplum-db/gp-common-go-libs/gplog"
	"github.com/greenplum-db/gpdb/gp/idl"
	"google.golang.org/grpc"
)

type RpcReply struct {
	hostname, address string
}

/*
ConnectHostList connects to given hostlist on the agent port, returns map of agent-address and connection
if fails, returns error.
*/
func (s *Server) ConnectHostList(hostList []string) (map[string]idl.AgentClient, error) {
	addressConnectionMap := make(map[string]idl.AgentClient)
	ctx, cancelFunc := context.WithTimeout(context.Background(), DialTimeout)
	defer cancelFunc()

	credentials, err := s.Credentials.LoadClientCredentials()
	if err != nil {
		cancelFunc()
		return nil, err
	}

	for _, address := range hostList {
		// Dial to the address
		if _, ok := addressConnectionMap[address]; !ok {
			remoteAddress := fmt.Sprintf("%s:%d", address, s.AgentPort)
			opts := []grpc.DialOption{
				grpc.WithBlock(),
				grpc.WithTransportCredentials(credentials),
				grpc.WithReturnConnectionError(),
			}
			if s.grpcDialer != nil {
				opts = append(opts, grpc.WithContextDialer(s.grpcDialer))
			}
			conn, err := grpc.DialContext(ctx, remoteAddress, opts...)
			if err != nil {
				cancelFunc()
				return nil, fmt.Errorf("could not connect to agent on host %s: %w", address, err)
			}
			addressConnectionMap[address] = idl.NewAgentClient(conn)
		}
	}

	return addressConnectionMap, nil
}
func (s *Server) GetAllHostNames(ctx context.Context, request *idl.GetAllHostNamesRequest) (*idl.GetAllHostNamesReply, error) {
	gplog.Debug("Starting with rpc GetAllHostNames")
	addressConnectionMap, err := s.ConnectHostList(request.HostList)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	errs := make(chan error, len(addressConnectionMap))
	replies := make(chan RpcReply, len(addressConnectionMap))
	for address, conn := range addressConnectionMap {
		wg.Add(1)

		go func(addr string, connection idl.AgentClient) {
			defer wg.Done()
			request := idl.GetHostNameRequest{}
			reply, err := connection.GetHostName(context.Background(), &request)
			if err != nil {
				errs <- fmt.Errorf("host: %s, %w", addr, err)
				errs <- utils.LogAndReturnError(fmt.Errorf("getting hostname for %s failed with error:%v", addr, err))
				return
			}

			result := new(RpcReply)
			result.address = addr
			result.hostname = reply.Hostname
			replies <- *result
		}(address, conn)
	}
	wg.Wait()
	close(replies)
	close(errs)

	// Check for errors
	if len(errs) > 0 {
		for e := range errs {
			err = e
			break
		}
		return &idl.GetAllHostNamesReply{}, err
	}

	// Extract replies and populate reply
	hostNameMap := make(map[string]string)
	for reply := range replies {
		hostNameMap[reply.address] = reply.hostname
	}

	return &idl.GetAllHostNamesReply{HostNameMap: hostNameMap}, nil
}
