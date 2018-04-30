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

// Get returns Instance(s) from the repo, optionally with status information from Kubernetes
func (server *instancesServer) Get(ctx context.Context, getOptions *pb.GetOptions) (*pb.InstanceList, error) {
	request := NewGetRequest(getOptions, server.instancesDir)
	err := request.Run()
	if err != nil {
		return request.InstanceList, err
	}
	return request.InstanceList, nil
}

// Delete deletes an Instance from Kubernetes and then from the repo
func (server *instancesServer) Delete(ctx context.Context, pbInstance *pb.Instance) (*pb.Instance, error) {
	instance := NewInstance(pbInstance, server.instancesDir)
	// TODO put a mutex Lock around this?
	err := instance.delete()
	if err != nil {
		return pbInstance, err
	}
	return instance.Instance, nil
}
