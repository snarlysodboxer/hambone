package main

import (
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/plugins/state/memory"
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

func TestCreateSpecGroup(t *testing.T) {
	store := main.NewStore()

	// Can create SpecGroups with unique names
	specGroup := getSpecGroup("my-product")
	name, err := store.CreateSpecGroup(specGroup)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if name != specGroup.Name {
		t.Errorf("Mismatched names, got: '%s', want: '%s'", name, specGroup.Name)
		t.FailNow()
	}
	specGroup.Name = "my-product2"
	_, err = store.CreateSpecGroup(specGroup)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// Cannot create SpecGroup with empty name
	specGroup.Name = ""
	_, err = store.CreateSpecGroup(specGroup)
	if err == nil {
		t.Error("Received no error creating SpecGroup with empty name")
		t.FailNow()
	}
	msg := "SpecGroup Name cannot be empty"
	if err.Error() != msg {
		t.Errorf("Expected a different error, want: \"%s\", got: \"%s\"", msg, err.Error())
		t.FailNow()
	}
	specGroups, err := store.ListSpecGroups()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if len(specGroups) != 2 {
		t.Errorf("Expected a different number of SpecGroups, want: \"2\", got: \"%d\"", len(specGroups))
		t.FailNow()
	}

	// Cannot create SpecGroup with existing name
	specGroup.Name = "my-product"
	_, err = store.CreateSpecGroup(specGroup)
	if err == nil {
		t.Error("Received no error creating SpecGroup with existing name")
		t.FailNow()
	}
	msg = "A SpecGroup named 'my-product' already exists"
	if err.Error() != msg {
		t.Errorf("Expected a different error, want: \"%s\", got: \"%s\"", msg, err.Error())
		t.FailNow()
	}
	specGroups, err = store.ListSpecGroups()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if len(specGroups) != 2 {
		t.Errorf("Expected a different number of SpecGroups, want: \"%d\", got: \"%d\"", 2, len(specGroups))
		t.FailNow()
	}

	// Cannot create SpecGroup with non-unique Spec names
	specGroup.Name = "my-product3"
	specGroup.Specs[0].Name = "Service"
	_, err = store.CreateSpecGroup(specGroup)
	if err == nil {
		t.Error("Received no error creating SpecGroup with non-unique Spec names")
		t.FailNow()
	}
	msg = "Spec names are not unique"
	if err.Error() != msg {
		t.Errorf("Expected a different error, want: \"%s\", got: \"%s\"", msg, err.Error())
		t.FailNow()
	}
}

func TestCreateInstance(t *testing.T) {
	store := main.NewStore()

	// Cannot create Instance referencing SpecGroup which doesn't exist
	instance := &pb.Instance{
		Name: "my-client", SpecGroupName: "my-product", ValueSets: []*pb.ValueSet{
			&pb.ValueSet{"Deployment", `{"Name": "my-client"}`},
			&pb.ValueSet{"Service", `{"Name": "my-client"}`},
		},
	}
	_, err := store.CreateInstance(instance)
	if err == nil {
		t.Error("Received no error creating Instance referencing non-existent SpecGroup")
		t.FailNow()
	}
	msg := "No SpecGroup exists named 'my-product'"
	if err.Error() != msg {
		t.Errorf("Expected a different error, want: \"%s\", got: \"%s\"", msg, err.Error())
		t.FailNow()
	}
	instances, err := store.ListInstances()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if len(instances) != 0 {
		t.Errorf("Expected a different number of Instances, want: \"0\", got: \"%d\"", len(instances))
		t.FailNow()
	}

	// Cannot create Instance referencing Specs which don't exist (in existing SpecGroup)
	specGroup := getSpecGroup("my-product")
	_, err = store.CreateSpecGroup(specGroup)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	instance.ValueSets[0].SpecName = "NonExistent"
	_, err = store.CreateInstance(instance)
	if err == nil {
		t.Error("Received no error creating Instance referencing non-existent Specs in existing SpecGroup")
		t.FailNow()
	}
	msg = "SpecGroup 'my-product' has no Spec named 'NonExistent'"
	if err.Error() != msg {
		t.Errorf("Expected a different error, want: \"%s\", got: \"%s\"", msg, err.Error())
		t.FailNow()
	}

	// Cannot create Instances with empty Names
	instance.ValueSets[0].SpecName = "Deployment"
	instance.Name = ""
	_, err = store.CreateInstance(instance)
	if err == nil {
		t.Error("Received no error creating Instance with empty Name")
		t.FailNow()
	}
	msg = "Instance Name cannot be empty"
	if err.Error() != msg {
		t.Errorf("Expected a different error, want: \"%s\", got: \"%s\"", msg, err.Error())
		t.FailNow()
	}

	// Can create valid Instances
	instance.Name = "my-client"
	name, err := store.CreateInstance(instance)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if name != instance.Name {
		t.Errorf("Mismatched names, got: '%s', want: '%s'", name, instance.Name)
		t.FailNow()
	}
	instance.Name = "my-client2"
	_, err = store.CreateInstance(instance)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	instances, err = store.ListInstances()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if len(instances) != 2 {
		t.Errorf("Expected a different number of Instances, want: \"2\", got: \"%d\"", len(instances))
		t.FailNow()
	}

	// Cannot create Instance with existing Name
	instance.Name = "my-client2"
	_, err = store.CreateInstance(instance)
	if err == nil {
		t.Error("Received no error creating Instance with existing Name")
		t.FailNow()
	}
	msg = "An Instance named 'my-client2' already exists" // TODO
	if err.Error() != msg {
		t.Errorf("Expected a different error, want: \"%s\", got: \"%s\"", msg, err.Error())
		t.FailNow()
	}

	// Cannot create Instance with non-unique ValueSet SpecNames
	instance.Name = "my-client3"
	instance.ValueSets[0].SpecName = "Service"
	_, err = store.CreateInstance(instance)
	if err == nil {
		t.Error("Received no error creating Instance with non-unique ValueSet SpecNames")
		t.FailNow()
	}
	msg = "ValueSet SpecNames must be unique within an Instance"
	if err.Error() != msg {
		t.Errorf("Expected a different error, want: \"%s\", got: \"%s\"", msg, err.Error())
		t.FailNow()
	}

	// Cannot create Instance with empty SpecName in ValueSet
	instance.ValueSets[0].SpecName = ""
	_, err = store.CreateInstance(instance)
	if err == nil {
		t.Error("Received no error creating Instance with ValueSet with empty SpecName")
		t.FailNow()
	}
	msg = "SpecGroup 'my-product' has no Spec named ''"
	if err.Error() != msg {
		t.Errorf("Expected a different error, want: \"%s\", got: \"%s\"", msg, err.Error())
		t.FailNow()
	}

	// Cannot create Instance with empty JsonBlob in ValueSet
	instance.ValueSets[0].SpecName = "Deployment"
	instance.ValueSets[0].JsonBlob = ""
	_, err = store.CreateInstance(instance)
	if err == nil {
		t.Error("Received no error creating Instance with ValueSet with empty JsonBlob")
		t.FailNow()
	}
	msg = "ValueSet JsonBlob's must be non-empty"
	if err.Error() != msg {
		t.Errorf("Expected a different error, want: \"%s\", got: \"%s\"", msg, err.Error())
		t.FailNow()
	}
}
