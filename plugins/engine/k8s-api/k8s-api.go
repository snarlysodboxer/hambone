package main

import (
	"fmt"
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/plugins/render"
)

var K8sEngine K8sAPIEngine

type K8sAPIEngine struct {
	renderer render.Interface
}

func (engine *K8sAPIEngine) SetRenderer(renderer render.Interface) {
	engine.renderer = renderer
}

// Apply renders and applies
func (engine *K8sAPIEngine) ApplyInstance(instance *pb.Instance) error {
	rendered, err := engine.renderer.Render(instance)
	if err != nil {
		return err
	}
	for _, r := range rendered {
		fmt.Printf("Applying the following spec:\n%s\n", r)
	}
	return nil
}
