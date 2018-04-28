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
func (server *instancesServer) Apply(ctx context.Context, instance *pb.Instance) (*pb.Instance, error) {
	// TODO put a mutex Lock around this?
	instance, err := server.apply(instance)
	if err != nil {
		return instance, err
	}
	return instance, nil
}
