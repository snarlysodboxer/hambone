package instances

import (
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/pkg/state"
	"golang.org/x/net/context"
)

type InstancesServer struct {
	InstancesDir string
	StateStore   state.StateEngine
}

func NewInstancesServer(instancesDir string, stateStore state.StateEngine) *InstancesServer {
	return &InstancesServer{instancesDir, stateStore}
}

// Apply adds/updates the given Instance, applies it to Kubernetes, and commits the changes, rolling back as necessary
func (server *InstancesServer) Apply(ctx context.Context, pbInstance *pb.Instance) (*pb.Instance, error) {
	instance := NewInstance(pbInstance, server)
	err := instance.apply()
	if err != nil {
		return pbInstance, err
	}
	return instance.Instance, nil
}

// Get returns Instance(s) from the repo, optionally with status information from Kubernetes
func (server *InstancesServer) Get(ctx context.Context, getOptions *pb.GetOptions) (*pb.InstanceList, error) {
	request := NewGetRequest(getOptions, server)
	err := request.Run()
	if err != nil {
		return request.InstanceList, err
	}
	return request.InstanceList, nil
}

// Delete deletes an Instance from Kubernetes and then from the repo
func (server *InstancesServer) Delete(ctx context.Context, pbInstance *pb.Instance) (*pb.Instance, error) {
	instance := NewInstance(pbInstance, server)
	err := instance.delete()
	if err != nil {
		return pbInstance, err
	}
	return instance.Instance, nil
}
