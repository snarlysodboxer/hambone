package instances

import (
	pb "github.com/snarlysodboxer/hambone/generated"
	"golang.org/x/net/context"
)

type instancesServer struct {
	instancesDir string
}

func NewInstancesServer(instancesDir string) *instancesServer {
	return &instancesServer{instancesDir}
}

// Apply adds/updates the given Instance, applies it to Kubernetes, and commits the changes, rolling back as necessary
func (server *instancesServer) Apply(ctx context.Context, pbInstance *pb.Instance) (*pb.Instance, error) {
	instance := NewInstance(pbInstance, server.instancesDir)
	// TODO put a mutex Lock around this?
	err := instance.apply()
	if err != nil {
		return pbInstance, err
	}
	return instance.Instance, nil
}
