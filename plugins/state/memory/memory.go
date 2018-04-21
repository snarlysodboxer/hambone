package main

// This plugin is not concurrency safe

import (
	"fmt"
	pb "github.com/snarlysodboxer/hambone/generated"
)

var (
	StateStore MemoryStore
)

type MemoryStore struct {
	specGroups map[string]*pb.SpecGroup
	instances  map[string]*pb.Instance
}

func (store *MemoryStore) Init() {
	store.specGroups = make(map[string]*pb.SpecGroup)
	store.instances = make(map[string]*pb.Instance)
}

func NewStore() *MemoryStore {
	return &MemoryStore{make(map[string]*pb.SpecGroup), make(map[string]*pb.Instance)}
}

func (store *MemoryStore) instanceNameIsUnique(name string) bool {
	names := []string{}
	for instanceName, _ := range store.instances {
		names = append(names, instanceName)
	}
	names = append(names, name)
	if hasDuplicates(names) {
		return false
	}
	return true
}

func (store *MemoryStore) CreateInstance(instance *pb.Instance) (string, error) {
	// Ensure Instance Name is set
	if instance.Name == "" {
		return "", fmt.Errorf("Instance Name cannot be empty")
	}
	// Ensure Instance Name doesn't already exist
	if !store.instanceNameIsUnique(instance.Name) {
		return "", fmt.Errorf("An Instance named '%s' already exists", instance.Name)
	}
	// Ensure referenced SpecGroup Name exists
	if !store.specGroupExists(instance.SpecGroupName) {
		return "", fmt.Errorf("No SpecGroup exists named '%s'", instance.SpecGroupName)
	}
	// Ensure each ValueSet SpecName exists
	for _, valueSet := range instance.ValueSets {
		if !store.specExistsInSpecGroup(valueSet.SpecName, instance.SpecGroupName) {
			return "", fmt.Errorf("SpecGroup '%s' has no Spec named '%s'", instance.SpecGroupName, valueSet.SpecName)
		}
	}
	// Ensure each ValueSet SpecName is unique
	names := []string{}
	for _, valueSet := range instance.ValueSets {
		names = append(names, valueSet.SpecName)
	}
	if hasDuplicates(names) {
		return "", fmt.Errorf("ValueSet SpecNames must be unique within an Instance")
	}
	// Ensure each ValueSet JsonBlob is not empty
	for _, valueSet := range instance.ValueSets {
		if valueSet.JsonBlob == "" {
			return "", fmt.Errorf("ValueSet JsonBlobs must be non-empty")
		}
	}
	store.instances[instance.Name] = instance
	return instance.Name, nil
}

func (store *MemoryStore) ReadInstance(name string) (*pb.Instance, error) {
	found := false
	instance := &pb.Instance{}
	for instanceName, i := range store.instances {
		if instanceName == name {
			instance = i
			found = true
			break
		}
	}
	if !found {
		return instance, fmt.Errorf("Instance '%s' not found", name)
	}
	return instance, nil
}

func (store *MemoryStore) ListInstances() (map[string]string, error) {
	response := map[string]string{}
	for name, instance := range store.instances {
		response[name] = instance.SpecGroupName
	}
	return response, nil
}

func (store *MemoryStore) UpdateInstance(instance *pb.Instance) (string, error) {
	// Ensure Name is not empty
	if instance.Name == "" {
		return "", fmt.Errorf("Instance Name cannot be empty")
	}
	// Ensure each ValueSet SpecName is unique
	if !store.valueSetNamesAreUnique(instance.ValueSets) {
		return "", fmt.Errorf("ValueSet SpecNames are not unique")
	}
	// Ensure referenced SpecGroup exists
	if !store.specGroupExists(instance.SpecGroupName) {
		return "", fmt.Errorf("No SpecGroup exists named '%s'", instance.SpecGroupName)
	}
	// Ensure each ValueSet SpecName exists in SpecGroup
	for _, valueSet := range instance.ValueSets {
		if !store.specExistsInSpecGroup(valueSet.SpecName, instance.SpecGroupName) {
			return "", fmt.Errorf("SpecGroup '%s' has no Spec named '%s'", instance.SpecGroupName, valueSet.SpecName)
		}
	}
	// Ensure each ValueSet JsonBlob is not empty
	for _, valueSet := range instance.ValueSets {
		if valueSet.JsonBlob == "" {
			return "", fmt.Errorf("ValueSet JsonBlobs must be non-empty")
		}
	}

	store.instances[instance.Name] = instance
	return instance.Name, nil
}

func (store *MemoryStore) specGroupNameIsUnique(name string) bool {
	if _, ok := store.specGroups[name]; ok {
		return false
	}
	return true
}

func (store *MemoryStore) specNamesAreUnique(specs []*pb.Spec) bool {
	names := []string{}
	for _, spec := range specs {
		names = append(names, spec.Name)
	}
	if hasDuplicates(names) {
		return false
	}
	return true
}

func (store *MemoryStore) valueSetNamesAreUnique(valueSets []*pb.ValueSet) bool {
	names := []string{}
	for _, valueSet := range valueSets {
		names = append(names, valueSet.SpecName)
	}
	if hasDuplicates(names) {
		return false
	}
	return true
}

func uniqueNonEmptyElementsOf(s []string) []string {
	uniqueMap := make(map[string]bool, len(s))
	uniqueSlice := make([]string, len(uniqueMap))
	for _, element := range s {
		// if len(element) != 0 {
		if !uniqueMap[element] {
			uniqueSlice = append(uniqueSlice, element)
			uniqueMap[element] = true
		}
		// }
	}
	return uniqueSlice
}

func hasDuplicates(names []string) bool {
	if len(names) == len(uniqueNonEmptyElementsOf(names)) {
		return false
	}
	return true
}

func (store *MemoryStore) CreateSpecGroup(specGroup *pb.SpecGroup) (string, error) {
	// Ensure Name is not empty
	if specGroup.Name == "" {
		return "", fmt.Errorf("SpecGroup Name cannot be empty")
	}
	// Ensure Name is unique
	if !store.specGroupNameIsUnique(specGroup.Name) {
		return "", fmt.Errorf("A SpecGroup named '%s' already exists", specGroup.Name)
	}
	// Ensure each Spec Name is unique
	if !store.specNamesAreUnique(specGroup.Specs) {
		return "", fmt.Errorf("Spec names are not unique")
	}
	store.specGroups[specGroup.Name] = specGroup
	return specGroup.Name, nil
}

func (store *MemoryStore) ReadSpecGroup(name string) (*pb.SpecGroup, error) {
	found := false
	specGroup := &pb.SpecGroup{}
	for specGroupName, sG := range store.specGroups {
		if specGroupName == name {
			specGroup = sG
			found = true
			break
		}
	}
	if !found {
		return specGroup, fmt.Errorf("SpecGroup '%s' not found", name)
	}
	return specGroup, nil
}

func (store *MemoryStore) UpdateSpecGroup(specGroup *pb.SpecGroup) (string, error) {
	// Ensure Name is not empty
	if specGroup.Name == "" {
		return "", fmt.Errorf("SpecGroup Name cannot be empty")
	}
	// Ensure each Spec Name is unique
	if !store.specNamesAreUnique(specGroup.Specs) {
		return "", fmt.Errorf("Spec names are not unique")
	}

	store.specGroups[specGroup.Name] = specGroup
	return specGroup.Name, nil
}

func (store *MemoryStore) ListSpecGroups() ([]string, error) {
	names := []string{}
	for name, _ := range store.specGroups {
		names = append(names, name)
	}
	return names, nil
}

func (store *MemoryStore) specGroupExists(name string) bool {
	for specGroupName, _ := range store.specGroups {
		if specGroupName == name {
			return true
		}
	}
	return false
}

func (store *MemoryStore) specExistsInSpecGroup(specName, specGroupName string) bool {
	for sGName, specGroup := range store.specGroups {
		if sGName == specGroupName {
			for _, spec := range specGroup.Specs {
				if spec.Name == specName {
					return true
				}
			}
		}
	}
	return false
}
