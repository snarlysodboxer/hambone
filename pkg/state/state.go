package state

import (
	pb "github.com/snarlysodboxer/hambone/generated"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// InstanceNoExistError indicates an OldInstance was passed in the request, but there's no existing Instance to compare it to in the State Store.
	InstanceNoExistError = status.Error(codes.FailedPrecondition, "OldInstance was passed, but there's no existing Instance")
	// OldInstanceDiffersError indicates the request's OldInstance differs from the existing Instance. (Stale read.) Re-read and try again.
	OldInstanceDiffersError = status.Error(codes.FailedPrecondition, "OldInstance differs from existing Instance")
)

// Updater is the interface that enables updating Instances
type Updater interface {
	Init() error
	Cancel(error) error
	Commit(bool) error
	RunCleanupFuncs() error
}

// Deleter is the interface that enables deleting Instances
type Deleter interface {
	Init() error
	Cancel(error) error
	Commit() error
	RunCleanupFuncs() error
}

// Getter is the interface that enables getting Instances
type Getter interface {
	Run() error
	RunCleanupFuncs() error
}

// TemplatesGetter is the interface that enables getting Instance Templates
type TemplatesGetter interface {
	Run() error
	RunCleanupFuncs() error
}

// Engine is the interface that enables modifying state
type Engine interface {
	Init() error
	NewUpdater(*pb.Instance, string) Updater
	NewDeleter(*pb.Instance, string) Deleter
	NewGetter(*pb.GetOptions, *pb.InstanceList, string) Getter
	NewTemplatesGetter(*pb.InstanceList, string) TemplatesGetter
}
