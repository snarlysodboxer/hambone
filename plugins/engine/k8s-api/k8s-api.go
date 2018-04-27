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
	rendereds, err := engine.renderer.Render(instance)
	if err != nil {
		return err
	}
	for _, rendered := range rendereds {
		fmt.Printf("Applying the following spec:\n%s\n", rendered)
		// TODO Actually apply here
	}
	return nil
}

// Delete renders and deletes
func (engine *K8sAPIEngine) DeleteInstance(instance *pb.Instance) error {
	rendereds, err := engine.renderer.Render(instance)
	if err != nil {
		return err
	}
	// TODO Actually delete here
	for _, rendered := range rendereds {
		fmt.Printf("Deleting the following spec:\n%s\n", rendered)
	}
	return nil
}

// Status returns status information
// func (engine *K8sAPIEngine) StatusInstance(instance *pb.Instance) (*pb.StatusMessage, error) {
//     // TODO Actually get status here
//     for _, rendered := range rendereds {
//         fmt.Printf("Status:\n%s\n", "Fake Status")
//     }
//     return nil
// }
