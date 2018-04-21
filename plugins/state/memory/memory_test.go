package main

import (
	"github.com/golang/protobuf/proto"
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

func getInstance(name string) *pb.Instance {
	instance := &pb.Instance{
		Name: name, SpecGroupName: "my-product", ValueSets: []*pb.ValueSet{
			&pb.ValueSet{"Deployment", `{"Name": "my-client"}`},
			&pb.ValueSet{"Service", `{"Name": "my-client"}`},
		},
	}
	return instance
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

func TestReadSpecGroup(t *testing.T) {
	store := main.NewStore()
	sG := getSpecGroup("my-product")
	name, err := store.CreateSpecGroup(sG)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// Reads existing SpecGroup
	sGNew := getSpecGroup("my-product")
	specGroup, err := store.ReadSpecGroup(name)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if !proto.Equal(specGroup, sGNew) {
		t.Errorf("Created and Read objects are not equal, created: '%v', read: '%v'", sGNew, specGroup)
		t.FailNow()
	}

	// Errors Reading non-existent SpecGroup
	specGroup, err = store.ReadSpecGroup("asdf")
	if err == nil {
		t.Error("Received no error Reading non-existent SpecGroup")
		t.FailNow()
	}
	msg := "SpecGroup 'asdf' not found"
	if err.Error() != msg {
		t.Errorf("Expected a different error, want: \"%s\", got: \"%s\"", msg, err.Error())
		t.FailNow()
	}
}

func TestListSpecGroups(t *testing.T) {
	store := main.NewStore()

	// Read SpecGroups list
	specGroup := getSpecGroup("my-product")
	_, err := store.CreateSpecGroup(specGroup)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	specGroup1 := getSpecGroup("my-product1")
	_, err = store.CreateSpecGroup(specGroup1)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	specGroups, err := store.ListSpecGroups()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	expected := []string{"my-product", "my-product1"}
	if specGroups[0] != "my-product" || specGroups[1] != "my-product1" {
		t.Errorf("Expected a different SpecGroups list, want: \"%v\", got: \"%v\"", expected, specGroups)
		t.FailNow()
	}
}

func TestCreateInstance(t *testing.T) {
	store := main.NewStore()

	// Cannot create Instance referencing SpecGroup which doesn't exist
	instance := getInstance("my-client")
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

func TestReadInstance(t *testing.T) {
	store := main.NewStore()
	specGroup := getSpecGroup("my-product")
	_, err := store.CreateSpecGroup(specGroup)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	i := getInstance("my-client")
	name, err := store.CreateInstance(i)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// Reads existing Instance
	iNew := getInstance("my-client")
	instance, err := store.ReadInstance(name)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if !proto.Equal(instance, iNew) {
		t.Errorf("Created and Read objects are not equal, created: '%v', read: '%v'", iNew, instance)
		t.FailNow()
	}

	// Errors Reading non-existent Instance
	instance, err = store.ReadInstance("asdf")
	if err == nil {
		t.Error("Received no error Reading non-existent Instance")
		t.FailNow()
	}
	msg := "Instance 'asdf' not found"
	if err.Error() != msg {
		t.Errorf("Expected a different error, want: \"%s\", got: \"%s\"", msg, err.Error())
		t.FailNow()
	}
}

func TestListInstances(t *testing.T) {
	store := main.NewStore()
	specGroup := getSpecGroup("my-product")
	_, err := store.CreateSpecGroup(specGroup)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// Read Instances list
	instance := getInstance("my-client")
	_, err = store.CreateInstance(instance)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	instance1 := getInstance("my-client1")
	_, err = store.CreateInstance(instance1)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	instances, err := store.ListInstances()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	expected := map[string]string{"my-client": "my-product", "my-client1": "my-product1"}
	if instances["my-client"] != "my-product" || instances["my-client1"] != "my-product" {
		t.Errorf("Expected a different Instances list, want: \"%v\", got: \"%v\"", expected, instances)
		t.FailNow()
	}
}
