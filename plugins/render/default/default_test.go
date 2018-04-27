package main

import (
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/plugins/render/default"
	// "github.com/snarlysodboxer/hambone/plugins/state/memory"
	"testing"
)

const (
	template = `this text template for {{.Name}}`
)

func getSpecGroup(name string) *pb.SpecGroup {
	spec := &pb.Spec{Name: "Deployment", Template: template}
	specs := []*pb.Spec{}
	specs = append(specs, spec)
	spec = &pb.Spec{Name: "Service", Template: template}
	specs = append(specs, spec)
	specGroup := &pb.SpecGroup{}
	specGroup.Name = name
	specGroup.Specs = specs
	return specGroup
}

func getInstance(name string) *pb.Instance {
	instance := &pb.Instance{
		Name: name, SpecGroupName: "my-product", ValueSets: []*pb.ValueSet{
			&pb.ValueSet{"Deployment", `{"Name": "my-client"}`},
			&pb.ValueSet{"Service", `{"Name": "my-client"}`},
		},
	}
	return instance
}

func TestRenderInstance(t *testing.T) {
	store := main.NewStore()
	specGroup := getSpecGroup("my-product")
	name, err := store.CreateSpecGroup(specGroup)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	instance := getInstance("my-client")
	_, err = store.CreateInstance(instance)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	renderer := main.Renderer
	rendereds, err := renderer.Render(instance)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	want := `asfd`
	if rendereds[0] != want {
		t.Errorf("Expected a different rendered Spec, want: \"%s\", got: \"%s\"", want, rendereds[0])
		t.FailNow()
	}
}
