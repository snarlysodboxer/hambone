package main

import (
	"fmt"
	pb "github.com/snarlysodboxer/hambone/generated"
)

var K8sEngine K8sAPIEngine

type K8sAPIEngine struct {
}

// Apply renders and applies
func (engine *K8sAPIEngine) ApplyInstance(instance *pb.Instance) error {
	fmt.Println("k8s apply instance placeholder")
	return nil
}
