package engine

import (
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/plugins/render"
)

type Interface interface {
	SetRenderer(render.Interface)
	ApplyInstance(*pb.Instance) error
	DeleteInstance(*pb.Instance) error
	// StatusInstance(*pb.Instance) error
}
