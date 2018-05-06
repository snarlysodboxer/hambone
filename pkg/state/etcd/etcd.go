package etcd

import (
	"context"
	"errors"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/clientv3util"
	"github.com/coreos/etcd/clientv3/concurrency"
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/pkg/helpers"
	"github.com/snarlysodboxer/hambone/pkg/state"
	"time"
)

const (
	dialTimeout       = 5 * time.Second
	requestTimeout    = 5 * time.Second
	instanceKeyPrefix = "hambone_instance_"
)

var (
	StateStore EtcdEngine
	endpoints  = []string{"http://127.0.0.1:2379"} // TODO
)

type EtcdEngine struct{}

func (engine *EtcdEngine) Init() error {
	return nil
}

func (engine *EtcdEngine) NewGetter(options *pb.GetOptions, list *pb.InstanceList, instancesDir string) state.Getter {
	return &EtcdGetter{options, list, instancesDir}
}

func (engine *EtcdEngine) NewUpdater(instance *pb.Instance, instancesDir string) state.Updater {
	instanceDir, instanceFile := helpers.GetInstanceDirFile(instancesDir, instance.Name)
	return &EtcdUpdater{instance, instanceDir, instanceFile, nil, nil, nil}
}

func (engine *EtcdEngine) NewDeleter(instance *pb.Instance, instancesDir string) state.Deleter {
	instanceDir, instanceFile := helpers.GetInstanceDirFile(instancesDir, instance.Name)
	return &EtcdDeleter{instance, instanceDir, instanceFile, nil, nil, nil}
}

type EtcdGetter struct {
	*pb.GetOptions
	*pb.InstanceList
	instancesDir string
}

func (getter *EtcdGetter) Run() error {
	list := getter.InstanceList

	// setup client
	clientV3, err := clientv3.New(clientv3.Config{Endpoints: endpoints, DialTimeout: dialTimeout})
	if err != nil {
		return err
	}
	helpers.Println("Created clientV3")
	defer func() { clientV3.Close(); helpers.Println("Closed clientV3") }()
	kvClient := clientv3.NewKV(clientV3)

	// get key-values
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	response := &clientv3.GetResponse{}
	if getter.GetOptions.GetName() != "" {
		response, err = kvClient.Get(ctx, getInstanceKey(getter.GetOptions.GetName()))
	} else {
		response, err = kvClient.Get(ctx, instanceKeyPrefix, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))

	}
	cancel()
	if err != nil {
		return err
	}

	// load key-values into instanceList
	for _, kv := range response.Kvs {
		list.Instances = append(list.Instances, &pb.Instance{
			Name: stripInstanceKeyPrefix(string(kv.Key)), KustomizationYaml: string(kv.Value),
		})
	}

	if getter.GetOptions.GetName() == "" {
		// filter list to start and stop points in getOptions
		indexStart, indexStop := helpers.ConvertStartStopToSliceIndexes(getter.GetOptions.GetStart(), getter.GetOptions.GetStop(), int32(len(list.Instances)))
		if indexStop == 0 {
			list.Instances = list.Instances[indexStart:]
		} else {
			list.Instances = list.Instances[indexStart:indexStop]
		}
	}

	// mkdir and write file for each
	for _, instance := range list.Instances {
		_, instanceFile := helpers.GetInstanceDirFile(getter.instancesDir, instance.Name)
		if err = helpers.MkdirFile(instanceFile, instance.KustomizationYaml); err != nil {
			return err
		}
	}

	return nil
}

type EtcdUpdater struct {
	*pb.Instance
	instanceDir  string
	instanceFile string
	etcdClient   *clientv3.Client
	etcdSession  *concurrency.Session
	etcdMutex    *concurrency.Mutex
}

// Init is expected to do any init related to the state store,
//   as well as write the kustomization.yaml file
func (updater *EtcdUpdater) Init() error {
	instanceFile := updater.instanceFile
	instanceKey := getInstanceKey(updater.Instance.Name)

	// setup client
	clientV3, err := clientv3.New(clientv3.Config{Endpoints: endpoints, DialTimeout: dialTimeout})
	if err != nil {
		return err
	}
	helpers.Println("Created clientV3")
	updater.etcdClient = clientV3
	kvClient := clientv3.NewKV(clientV3)

	// take out an etcd lock
	session, mutex, err := getSessionAndMutex(clientV3, instanceKey)
	if err != nil {
		clientV3.Close()
		helpers.Println("Closed clientV3")
		return err
	}
	updater.etcdSession, updater.etcdMutex = session, mutex

	// ensure OldInstance passed in request equals current Instance in etcd
	if err := oldInstanceEqualsCurrentInstanceIfSet(kvClient, instanceKey, updater.Instance.GetOldInstance()); err != nil {
		return updater.cleanUp(err)
	}
	updater.Instance.OldInstance = nil // don't return OldInstance in response

	// mkdir and write file
	if err := helpers.MkdirFile(instanceFile, updater.Instance.KustomizationYaml); err != nil {
		return updater.cleanUp(err)
	}

	return nil
}

// Cancel is expected to clean up any mess, and remove the kustomization.yaml file/dir
func (updater *EtcdUpdater) Cancel(err error) error {
	return updater.cleanUp(err)
}

// Commit is expected to add/update the Instance in the state store
func (updater *EtcdUpdater) Commit() (erR error) {
	instanceKey := getInstanceKey(updater.Instance.Name)
	defer func() { erR = updater.cleanUp(erR) }()

	// at this point, OldInstance matches if present, we have a lock, and
	//   it doesn't matter if the key is pre-existing or not, so just put
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	_, err := updater.etcdClient.Put(ctx, instanceKey, updater.Instance.KustomizationYaml)
	cancel()
	if err != nil {
		return err
	}

	return nil

}

func (updater *EtcdUpdater) cleanUp(err error) error {
	return cleanUp(updater.etcdMutex, updater.etcdSession, updater.etcdClient, err)
}

type EtcdDeleter struct {
	*pb.Instance
	instanceDir  string
	instanceFile string
	// TODO shorten these names
	etcdClient  *clientv3.Client
	etcdSession *concurrency.Session
	etcdMutex   *concurrency.Mutex
}

// Init is expected to do any init related to the state store,
//   as well as write the kustomization.yaml file
func (deleter *EtcdDeleter) Init() error {
	instanceFile := deleter.instanceFile
	instanceKey := getInstanceKey(deleter.Instance.Name)

	// setup client
	clientV3, err := clientv3.New(clientv3.Config{Endpoints: endpoints, DialTimeout: dialTimeout})
	if err != nil {
		return err
	}
	helpers.Println("Created clientV3")
	deleter.etcdClient = clientV3
	kvClient := clientv3.NewKV(clientV3)

	// ensure Instance exists in etcd
	txnResponse, err := kvClient.Txn(context.Background()).
		If(clientv3util.KeyExists(instanceKey)).
		Commit()
	if err != nil {
		clientV3.Close()
		helpers.Println("Closed clientV3")
		return err
	}
	if !txnResponse.Succeeded { // if !key exists
		clientV3.Close()
		helpers.Println("Closed clientV3")
		return errors.New(fmt.Sprintf("No etcd key found for %s", deleter.Instance.Name))
	}

	// take out an etcd lock
	session, mutex, err := getSessionAndMutex(clientV3, instanceKey)
	if err != nil {
		clientV3.Close()
		helpers.Println("Closed clientV3")
		return err
	}
	deleter.etcdSession, deleter.etcdMutex = session, mutex

	// ensure passed OldInstance equals current Instance in etcd
	if err := oldInstanceEqualsCurrentInstanceIfSet(kvClient, instanceKey, deleter.Instance.GetOldInstance()); err != nil {
		return deleter.cleanUp(err)
	}
	deleter.Instance.OldInstance = nil // don't return OldInstance in response

	// load kustomizationYaml from etcd
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	response, err := kvClient.Get(ctx, instanceKey)
	cancel()
	if err != nil {
		return deleter.cleanUp(err)
	}
	if len(response.Kvs) != 1 {
		return errors.New(fmt.Sprintf("Expected 1 key-value set, got %d", len(response.Kvs)))
	}
	deleter.Instance.KustomizationYaml = string(response.Kvs[0].Value)

	// mkdir and write file
	if err := helpers.MkdirFile(instanceFile, deleter.Instance.KustomizationYaml); err != nil {
		return deleter.cleanUp(err)
	}

	return nil
}

// Cancel is expected to clean up any mess, and re-add the kustomization.yaml file/dir
func (deleter *EtcdDeleter) Cancel(err error) error {
	// TODO do we need to add file here?
	return deleter.cleanUp(err)
}

// Commit is expected to delete the Instance from the state store
func (deleter *EtcdDeleter) Commit() (erR error) {
	instanceKey := getInstanceKey(deleter.Instance.Name)
	defer func() { erR = deleter.cleanUp(erR) }()

	// at this point, OldInstance matches if present, we have a lock, and
	//   the key exists, so just delete
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	_, err := deleter.etcdClient.Delete(ctx, instanceKey, clientv3.WithPrefix())
	cancel()
	if err != nil {
		return err
	}
	return nil
}

func (deleter *EtcdDeleter) cleanUp(err error) error {
	return cleanUp(deleter.etcdMutex, deleter.etcdSession, deleter.etcdClient, err)
}

func getInstanceKey(name string) string {
	return fmt.Sprintf("%s%s", instanceKeyPrefix, name)
}

func stripInstanceKeyPrefix(name string) string {
	return name[len(instanceKeyPrefix):]
}

func getSessionAndMutex(clientV3 *clientv3.Client, instanceKey string) (*concurrency.Session, *concurrency.Mutex, error) {
	// TODO consider setting lease expiration
	session, err := concurrency.NewSession(clientV3)
	if err != nil {
		return &concurrency.Session{}, &concurrency.Mutex{}, err
	}
	helpers.Println("Created Concurrency session")

	mutex := concurrency.NewMutex(session, instanceKey)
	if err = mutex.Lock(context.TODO()); err != nil {
		session.Close()
		helpers.Println("Closed Concurrency session")
		return session, &concurrency.Mutex{}, err
	}
	helpers.Println("Obtained lock for", instanceKey)

	return session, mutex, nil
}

func cleanUp(mutex *concurrency.Mutex, session *concurrency.Session, clientV3 *clientv3.Client, err error) error {
	key := mutex.Key()
	if innerError := mutex.Unlock(context.TODO()); innerError != nil {
		if err != nil {
			session.Close()
			clientV3.Close()
			helpers.Println("Closed Concurrency Session and clientV3")
			return errors.New(fmt.Sprintf("ERROR releasing lock after another error: %s\nOriginal Error:\n%s\n", innerError.Error(), err.Error()))
		}
		session.Close()
		clientV3.Close()
		helpers.Println("Closed Concurrency Session and clientV3")
		return errors.New(fmt.Sprintf("ERROR releasing lock: %s\n", innerError.Error()))
	}
	session.Close()
	clientV3.Close()
	helpers.Printf("Released lock for %s, closed Concurrency Session and clientV3\n", key)

	return err
}

func oldInstanceEqualsCurrentInstanceIfSet(kvClient clientv3.KV, instanceKey string, oldInstance *pb.Instance) error {
	if oldInstance != nil {
		txnResponse, err := kvClient.Txn(context.Background()).
			If(clientv3util.KeyExists(instanceKey)).
			Commit()
		if err != nil {
			return err
		}
		if txnResponse.Succeeded { // if key exists
			// check for modified since read
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			response, err := kvClient.Get(ctx, instanceKey)
			cancel()
			if err != nil {
				return err
			}
			if len(response.Kvs) != 1 {
				return errors.New(fmt.Sprintf("Expected 1 key-value set, got %d", len(response.Kvs)))
			}
			currentValue := response.Kvs[0].Value
			if string(currentValue) != oldInstance.KustomizationYaml {
				return errors.New("This Instance has been modified since you read it!")
			}
		}
	}

	return nil
}
