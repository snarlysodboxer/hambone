package engine

import (
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/plugins/render"
)

type Interface interface {
	Init(render.Interface, string, bool) error
	ApplyInstance(*pb.Instance) error
	DeleteInstance(*pb.Instance) error
	// StatusInstance(*pb.Instance) error
}
