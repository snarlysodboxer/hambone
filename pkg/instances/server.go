package instances

import (
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/pkg/state"
	"golang.org/x/net/context"
)

// Server is the interface for interacting with Instances
type Server struct {
	InstancesDir         string
	TemplatesDir         string
	StateStore           state.Engine
	EnableKustomizeBuild bool
	EnableKubectl        bool
}

// NewInstancesServer returns an instantiated InstanceServer
func NewInstancesServer(instancesDir, templatesDir string, stateStore state.Engine, enableKustomizeBuild, enableKubectl bool) *Server {
	return &Server{instancesDir, templatesDir, stateStore, enableKustomizeBuild, enableKubectl}
}

// Apply adds/updates the given Instance, applies it to Kubernetes if desired, and commits the changes, rolling back as necessary
func (server *Server) Apply(ctx context.Context, pbInstance *pb.Instance) (*pb.Instance, error) {
	instance := NewInstance(pbInstance, server)
	err := instance.Apply(false)
	if err != nil {
		return pbInstance, err
	}

	return instance.Instance, nil
}

// Get returns Instance(s) from the State Store, optionally with status information from Kubernetes
func (server *Server) Get(ctx context.Context, getOptions *pb.GetOptions) (*pb.InstanceList, error) {
	request := NewGetRequest(getOptions, server)
	err := request.Run()
	if err != nil {
		return request.InstanceList, err
	}

	return request.InstanceList, nil
}

// Delete deletes an Instance from Kubernetes and then from the State Store
func (server *Server) Delete(ctx context.Context, pbInstance *pb.Instance) (*pb.Instance, error) {
	instance := NewInstance(pbInstance, server)
	err := instance.Delete()
	if err != nil {
		return pbInstance, err
	}

	return instance.Instance, nil
}

// GetTemplates returns a list of Instance templates from the State Store
func (server *Server) GetTemplates(ctx context.Context, _ *pb.Empty) (*pb.InstanceList, error) {
	request := NewGetTemplatesRequest(server)
	err := request.Run()
	if err != nil {
		return request.InstanceList, err
	}

	return request.InstanceList, nil
}
