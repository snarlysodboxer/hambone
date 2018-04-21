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

func expectInstances(quantity int, store *main.MemoryStore, t *testing.T) {
	instances, err := store.ListInstances()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if len(instances) != quantity {
		t.Errorf("Expected a different number of SpecGroups, want: \"%d\", got: \"%d\"", quantity, len(instances))
		t.FailNow()
	}
}

func expectSpecGroups(quantity int, store *main.MemoryStore, t *testing.T) {
	specGroups, err := store.ListSpecGroups()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if len(specGroups) != quantity {
		t.Errorf("Expected a different number of SpecGroups, want: \"%d\", got: \"%d\"", quantity, len(specGroups))
		t.FailNow()
	}
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
	errorMsg := "Received no error creating SpecGroup with empty name"
	expectMsg := "SpecGroup Name cannot be empty"
	expectError(err, expectMsg, errorMsg, t)
	expectSpecGroups(2, store, t)

	// Cannot create SpecGroup with existing name
	specGroup.Name = "my-product"
	_, err = store.CreateSpecGroup(specGroup)
	errorMsg = "Received no error creating SpecGroup with existing name"
	expectMsg = "A SpecGroup named 'my-product' already exists"
	expectError(err, expectMsg, errorMsg, t)
	expectSpecGroups(2, store, t)

	// Cannot create SpecGroup with non-unique Spec names
	specGroup.Name = "my-product3"
	specGroup.Specs[0].Name = "Service"
	_, err = store.CreateSpecGroup(specGroup)
	errorMsg = "Received no error creating SpecGroup with non-unique Spec names"
	expectMsg = "Spec names are not unique"
	expectError(err, expectMsg, errorMsg, t)
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
	errorMsg := "Received no error Reading non-existent SpecGroup"
	expectMsg := "SpecGroup 'asdf' not found"
	expectError(err, expectMsg, errorMsg, t)
}

func TestListSpecGroups(t *testing.T) {
	store := main.NewStore()

	// Reads SpecGroups list
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

func TestUpdateSpecGroup(t *testing.T) {
	store := main.NewStore()
	specGroup := getSpecGroup("my-product")
	_, err := store.CreateSpecGroup(specGroup)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// Updates SpecGroup
	specGroup.Specs[0].Name = "NewDeployment"
	name, err := store.UpdateSpecGroup(specGroup)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	specGroupRead, err := store.ReadSpecGroup(name)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	specGroupCopy := getSpecGroup("my-product")
	specGroupCopy.Specs[0].Name = "NewDeployment"
	if !proto.Equal(specGroupCopy, specGroupRead) {
		t.Errorf("Expected an updated SpecGroup, want: \"%v\", got: \"%v\"", specGroupCopy, specGroupRead)
		t.FailNow()
	}

	// Cannot update SpecGroup with empty name
	specGroup.Name = ""
	_, err = store.UpdateSpecGroup(specGroup)
	errorMsg := "Received no error trying to update a SpecGroup with an empty name"
	expectMsg := "SpecGroup Name cannot be empty"
	expectError(err, expectMsg, errorMsg, t)

	expectSpecGroups(1, store, t)

	// Cannot update SpecGroup with non-unique Spec names
	specGroup.Name = "my-product"
	specGroup.Specs[0].Name = "Service"
	_, err = store.UpdateSpecGroup(specGroup)
	errorMsg = "Received no error updating a SpecGroup with non-unique Spec names"
	expectMsg = "Spec names are not unique"
	expectError(err, expectMsg, errorMsg, t)
}

//////// Instances

func TestCreateInstance(t *testing.T) {
	store := main.NewStore()

	// Cannot create Instance referencing SpecGroup which doesn't exist
	instance := getInstance("my-client")
	_, err := store.CreateInstance(instance)
	errorMsg := "Received no error creating Instance referencing non-existent SpecGroup"
	expectMsg := "No SpecGroup exists named 'my-product'"
	expectError(err, expectMsg, errorMsg, t)
	expectInstances(0, store, t)

	// Cannot create Instance referencing Specs which don't exist (in existing SpecGroup)
	specGroup := getSpecGroup("my-product")
	_, err = store.CreateSpecGroup(specGroup)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	instance.ValueSets[0].SpecName = "NonExistent"
	_, err = store.CreateInstance(instance)
	errorMsg = "Received no error creating Instance referencing non-existent Specs in existing SpecGroup"
	expectMsg = "SpecGroup 'my-product' has no Spec named 'NonExistent'"
	expectError(err, expectMsg, errorMsg, t)

	// Cannot create Instances with empty Names
	instance.ValueSets[0].SpecName = "Deployment"
	instance.Name = ""
	_, err = store.CreateInstance(instance)
	errorMsg = "Received no error creating Instance with empty Name"
	expectMsg = "Instance Name cannot be empty"
	expectError(err, expectMsg, errorMsg, t)

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
	instance1 := getInstance("my-client2")
	_, err = store.CreateInstance(instance1)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	expectInstances(2, store, t)

	// Cannot create Instance with existing Name
	instance.Name = "my-client2"
	_, err = store.CreateInstance(instance)
	errorMsg = "Received no error creating Instance with existing Name"
	expectMsg = "An Instance named 'my-client2' already exists"
	expectError(err, expectMsg, errorMsg, t)

	// Cannot create Instance with non-unique ValueSet SpecNames
	instance.Name = "my-client3"
	instance.ValueSets[0].SpecName = "Service"
	_, err = store.CreateInstance(instance)
	errorMsg = "Received no error creating Instance with non-unique ValueSet SpecNames"
	expectMsg = "ValueSet SpecNames must be unique within an Instance"
	expectError(err, expectMsg, errorMsg, t)

	// Cannot create Instance with empty SpecName in ValueSet
	instance.ValueSets[0].SpecName = ""
	_, err = store.CreateInstance(instance)
	errorMsg = "Received no error creating Instance with ValueSet with empty SpecName"
	expectMsg = "SpecGroup 'my-product' has no Spec named ''"
	expectError(err, expectMsg, errorMsg, t)

	// Cannot create Instance with empty JsonBlob in ValueSet
	instance.ValueSets[0].SpecName = "Deployment"
	instance.ValueSets[0].JsonBlob = ""
	_, err = store.CreateInstance(instance)
	errorMsg = "Received no error creating Instance with ValueSet with empty JsonBlob"
	expectMsg = "ValueSet JsonBlob's must be non-empty"
	expectError(err, expectMsg, errorMsg, t)
	expectInstances(2, store, t)
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

	// Errors when Reading non-existent Instance
	instance, err = store.ReadInstance("asdf")
	errorMsg := "Received no error Reading non-existent Instance"
	expectMsg := "Instance 'asdf' not found"
	expectError(err, expectMsg, errorMsg, t)
}

func TestListInstances(t *testing.T) {
	store := main.NewStore()
	specGroup := getSpecGroup("my-product")
	_, err := store.CreateSpecGroup(specGroup)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// Reads Instances list
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

func TestUpdateInstance(t *testing.T) {
	store := main.NewStore()
	specGroup := getSpecGroup("my-product")
	spec := &pb.Spec{Name: "NewDeployment", Template: template}
	specGroup.Specs = append(specGroup.Specs, spec)
	_, err := store.CreateSpecGroup(specGroup)
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

	// Updates Instance
	instance.ValueSets = append(instance.ValueSets, &pb.ValueSet{"NewDeployment", `{"Name": "my-client"}`})
	name, err := store.UpdateInstance(instance)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	instanceRead, err := store.ReadInstance(name)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	instanceCopy := getInstance("my-client")
	instanceCopy.ValueSets = append(instanceCopy.ValueSets, &pb.ValueSet{"NewDeployment", `{"Name": "my-client"}`})
	if !proto.Equal(instanceCopy, instanceRead) {
		t.Errorf("Expected an updated Instance, want:\n\"%v\", \ngot:\n\"%v\"", instanceCopy, instanceRead)
		t.FailNow()
	}

	// Cannot update Instance with empty name
	instance.Name = ""
	_, err = store.UpdateInstance(instance)
	errorMsg := "Received no error trying to update a Instance with an empty name"
	expectMsg := "Instance Name cannot be empty"
	expectError(err, expectMsg, errorMsg, t)

	// Cannot update Instance with ValueSets with non-unique SpecNames
	instance.Name = "my-client"
	instance.ValueSets[0].SpecName = "Service"
	_, err = store.UpdateInstance(instance)
	errorMsg = "Received no error updating a Instance with ValueSets with non-unique SpecNames"
	expectMsg = "ValueSet SpecNames are not unique"
	expectError(err, expectMsg, errorMsg, t)

	// Cannot update Instance referencing SpecGroup which doesn't exist
	// Cannot update Instance referencing Specs which don't exist (in existing SpecGroup)
	// Cannot update Instance with non-unique ValueSet SpecNames
	// Cannot update Instance with empty SpecName in ValueSet
	// Cannot update Instance with empty JsonBlob in ValueSet
}

func expectError(err error, expectMsg, errorMsg string, t *testing.T) {
	if err == nil {
		t.Error(errorMsg)
		t.FailNow()
	}
	if err.Error() != expectMsg {
		t.Errorf("Expected a different error, want: \"%s\", got: \"%s\"", expectMsg, err.Error())
		t.FailNow()
	}
}
