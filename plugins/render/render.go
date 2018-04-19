package render

import (
	pb "github.com/snarlysodboxer/hambone/generated"
)

type Interface interface {
	Render(*pb.Instance) ([]string, error)
}
