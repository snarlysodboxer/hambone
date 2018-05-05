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
	"io/ioutil"
	"os"
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
	return &EtcdDeleter{instance, instanceDir, instanceFile}
}

type EtcdGetter struct {
	*pb.GetOptions
	*pb.InstanceList
	instancesDir string
}

func (getter *EtcdGetter) Run() error {
	list := getter.InstanceList

	// setup client
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: dialTimeout,
	})
	if err != nil {
		return err
	}
	defer func() { helpers.Println("Close KV client"); client.Close() }()
	kvClient := clientv3.NewKV(client)
	helpers.Println("Created KV client")

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
	instanceDir := updater.instanceDir
	instanceKey := getInstanceKey(updater.Instance.Name)

	// setup client
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: dialTimeout,
	})
	if err != nil {
		return err
	}
	updater.etcdClient = client
	kvClient := clientv3.NewKV(updater.etcdClient)
	helpers.Println("Created KV client")

	// take out an etcd lock
	// TODO consider setting lease expiration
	session, err := concurrency.NewSession(updater.etcdClient)
	if err != nil {
		return err
	}
	updater.etcdSession = session
	helpers.Println("Created Concurrency session")
	mutex := concurrency.NewMutex(session, instanceKey)
	if err = mutex.Lock(context.TODO()); err != nil {
		return updater.cleanUp(err)
	}
	updater.etcdMutex = mutex
	helpers.Println("Obtained lock for", instanceKey)

	if updater.Instance.GetOldInstance() != nil {
		txnResponse, err := kvClient.Txn(context.Background()).
			If(clientv3util.KeyExists(instanceKey)).
			Commit()
		if err != nil {
			return updater.cleanUp(err)
		}
		if txnResponse.Succeeded { // if key exists
			// check for modified since read
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			response, err := kvClient.Get(ctx, instanceKey)
			cancel()
			if err != nil {
				return updater.cleanUp(err)
			}
			if len(response.Kvs) != 1 {
				erR := errors.New(fmt.Sprintf("Expected 1 key-value set, got %d", len(response.Kvs)))
				return updater.cleanUp(erR)
			}
			currentValue := response.Kvs[0].Value
			if string(currentValue) != updater.Instance.GetOldInstance().KustomizationYaml {
				erR := errors.New("This Instance has been modified since you read it!")
				return updater.cleanUp(erR)
			}
		}
		updater.Instance.OldInstance = nil
	}

	// mkdir
	if err := os.MkdirAll(instanceDir, 0755); err != nil {
		return updater.cleanUp(err)
	}

	// write <instancesDir>/<name>/kustomization.yaml
	if err := ioutil.WriteFile(instanceFile, []byte(updater.Instance.KustomizationYaml), 0644); err != nil {
		return updater.cleanUp(err)
	}
	helpers.Printf("Wrote `%s` with contents:\n\t%s\n", instanceFile, helpers.Indent([]byte(updater.Instance.KustomizationYaml)))

	return nil
}

// Cancel is expected to clean up any mess, and remove the kustomization.yaml file/dir
func (updater *EtcdUpdater) Cancel(err error) error {
	// TODO delete file (if not pre-existing?)

	return updater.cleanUp(err)
}

// Commit is expected to add/update the Instance in the state store
func (updater *EtcdUpdater) Commit() (erR error) {
	instanceKey := getInstanceKey(updater.Instance.Name)
	defer func() { erR = updater.cleanUp(nil) }() // TODO this swallows innerError from cleanUp, think about this

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
	if innerError := updater.etcdMutex.Unlock(context.TODO()); innerError != nil {
		if err != nil {
			updater.etcdSession.Close()
			updater.etcdClient.Close()
			return errors.New(fmt.Sprintf("ERROR releasing lock after another error: %s\nOriginal Error:\n%s\n", innerError.Error(), err.Error()))
		}
		updater.etcdSession.Close()
		updater.etcdClient.Close()
		return errors.New(fmt.Sprintf("ERROR releasing lock: %s\n", innerError.Error()))
	}
	updater.etcdSession.Close()
	updater.etcdClient.Close()
	helpers.Println("Released lock, closed session and client")
	return err
}

type EtcdDeleter struct {
	*pb.Instance
	instanceDir  string
	instanceFile string
}

// Init is expected to do any init related to the state store,
//   as well as write the kustomization.yaml file
func (deleter *EtcdDeleter) Init() error {
	instanceFile := deleter.instanceFile

	// ensure Instance exists
	if _, err := os.Stat(instanceFile); os.IsNotExist(err) {
		return errors.New(fmt.Sprintf("ERROR Instance not found at `%s`", instanceFile))
	}

	return nil
}

// Cancel is expected to clean up any mess, and re-add the kustomization.yaml file/dir
func (deleter *EtcdDeleter) Cancel(err error) error {
	return nil
}

// Commit is expected to delete the Instance from the state store
func (deleter *EtcdDeleter) Commit() error {
	// TODO consider the case where any of the following fail, but the objects have been deleted from k8s

	return nil
}

func getInstanceKey(name string) string {
	return fmt.Sprintf("%s%s", instanceKeyPrefix, name)
}

func stripInstanceKeyPrefix(name string) string {
	return name[len(instanceKeyPrefix):]
}
