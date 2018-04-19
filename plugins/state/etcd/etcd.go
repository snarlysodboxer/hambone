package main

import (
	"fmt"
	pb "github.com/snarlysodboxer/hambone/generated"
	"sort"
)

var StateStore EtcdStore

type EtcdStore struct {
}

func (store *EtcdStore) nextInstanceID() (int32, error) {
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

func (store *EtcdStore) AddInstance(instance *pb.Instance) (int32, error) {
	id, err := store.nextInstanceID()
	if err != nil {
		return 0, err
	}
	instance.Id = id
	fmt.Println("etcd store add Instance placeholder")
	return id, nil
}

func (store *EtcdStore) ListInstances() (map[int32]string, error) {
	fmt.Println("etcd store list Instances placeholder")
	return map[int32]string{}, nil
}

func (store *EtcdStore) nextSpecGroupID() (int32, error) {
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

func (store *EtcdStore) AddSpecGroup(specGroup *pb.SpecGroup) (int32, error) {
	id, err := store.nextSpecGroupID()
	if err != nil {
		return 0, err
	}
	specGroup.Id = id
	fmt.Println("etcd store add SpecGroup placeholder")
	return id, nil
}

func (store *EtcdStore) ListSpecGroups() (map[int32]string, error) {
	fmt.Println("etcd store list SpecGroups placeholder")
	return map[int32]string{}, nil
}
