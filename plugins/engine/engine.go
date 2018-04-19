package engine

import (
	pb "github.com/snarlysodboxer/hambone/generated"
)

type Interface interface {
	ApplyInstance(*pb.Instance) error
}
