package main

// This plugin is not concurrency safe

import (
	"fmt"
	pb "github.com/snarlysodboxer/hambone/generated"
	"sort"
)

var StateStore MemoryStore

type MemoryStore struct {
	specGroups []*pb.SpecGroup
	instances  []*pb.Instance
}

func (store *MemoryStore) NextInstanceID() (int32, error) {
	ids := []int{}
	list, err := store.ListInstances()
	if err != nil {
		return 0, err
	}
	if len(list) == 0 {
		return 1, nil
	}
	for key, _ := range list {
		ids = append(ids, int(key))
	}
	sort.Ints(ids)
	return int32(ids[len(ids)-1] + 1), nil
}

func (store *MemoryStore) CreateInstance(instance *pb.Instance) (int32, error) {
	id, err := store.NextInstanceID()
	if err != nil {
		return 0, err
	}
	instance.Id = id
	// Ensure specGroup id exists
	if !store.specGroupIDExists(instance.SpecGroupId) {
		return 0, fmt.Errorf("No SpecGroup exists with ID %d", instance.SpecGroupId)
	}
	// Ensure each spec id exists
	for _, valueSet := range instance.ValueSets {
		if !store.specIDExists(valueSet.SpecId, instance.SpecGroupId) {
			return 0, fmt.Errorf("No Spec exists with ID %d in SpecGroup %d", valueSet.SpecId, instance.SpecGroupId)
		}
	}
	store.instances = append(store.instances, instance)
	return id, nil
}

func (store *MemoryStore) ListInstances() (map[int32]string, error) {
	response := map[int32]string{}
	for _, instance := range store.instances {
		response[instance.Id] = instance.Name
	}
	return response, nil
}

func (store *MemoryStore) NextSpecGroupID() (int32, error) {
	ids := []int{}
	list, err := store.ListSpecGroups()
	if err != nil {
		return 0, err
	}
	if len(list) == 0 {
		return 1, nil
	}
	for key, _ := range list {
		ids = append(ids, int(key))
	}
	sort.Ints(ids)
	return int32(ids[len(ids)-1] + 1), nil
}

func (store *MemoryStore) CreateSpecGroup(specGroup *pb.SpecGroup) (int32, error) {
	id, err := store.NextSpecGroupID()
	if err != nil {
		return 0, err
	}
	specGroup.Id = id
	for sequence, spec := range specGroup.Specs {
		spec.Id = int32(sequence + 1)
	}
	store.specGroups = append(store.specGroups, specGroup)
	return id, nil
}

func (store *MemoryStore) ReadSpecGroup(id int32) (*pb.SpecGroup, error) {
	found := false
	specGroup := &pb.SpecGroup{}
	for _, sG := range store.specGroups {
		if sG.Id == id {
			specGroup = sG
			found = true
			break
		}
	}
	if !found {
		return specGroup, fmt.Errorf("SpecGroup %d not found", id)
	}
	return specGroup, nil
}

func (store *MemoryStore) ListSpecGroups() (map[int32]string, error) {
	response := map[int32]string{}
	for _, specGroup := range store.specGroups {
		response[specGroup.Id] = specGroup.Name
	}
	return response, nil
}

func (store *MemoryStore) specGroupIDExists(id int32) bool {
	for _, specGroup := range store.specGroups {
		if specGroup.Id == id {
			return true
		}
	}
	return false
}

func (store *MemoryStore) specIDExists(specID, specGroupID int32) bool {
	for _, specGroup := range store.specGroups {
		if specGroup.Id == specGroupID {
			for _, spec := range specGroup.Specs {
				if spec.Id == specID {
					return true
				}
			}
		}
	}
	return false
}
