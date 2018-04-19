package render

import (
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/plugins/state"
)

type Interface interface {
	SetStateStore(state.Interface)
	Render(*pb.Instance) ([]string, error)
}
