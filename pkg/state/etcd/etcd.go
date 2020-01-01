package etcd

import (
	"context"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/clientv3util"
	"github.com/coreos/etcd/clientv3/concurrency"
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/pkg/helpers"
	"github.com/snarlysodboxer/hambone/pkg/state"
	"log"
	"strings"
	"time"
)

const (
	dialTimeout       = 5 * time.Second
	requestTimeout    = 5 * time.Second
	instanceKeyPrefix = "hambone_instance_"
)

var (
	endpoints = []string{}
)

// Engine fulfills the StateStore interface
type Engine struct {
	EndpointsString string
}

// Init does setup
func (engine *Engine) Init() error {
	// parse and set endpoints
	endpoints = strings.Split(engine.EndpointsString, ",")
	return nil
}

// NewGetter returns an initialized Getter
func (engine *Engine) NewGetter(options *pb.GetOptions, list *pb.InstanceList, instancesDir string) state.Getter {
	return &etcdGetter{options, list, instancesDir}
}

// NewTemplatesGetter returns an initialized TemplatesGetter
func (engine *Engine) NewTemplatesGetter(list *pb.InstanceList, templatesDir string) state.TemplatesGetter {
	return &etcdTemplatesGetter{list, templatesDir}
}

// NewUpdater returns an initialized Updater
func (engine *Engine) NewUpdater(instance *pb.Instance, instancesDir string) state.Updater {
	instanceDir, instanceFile := helpers.GetInstanceDirFile(instancesDir, instance.Name)
	return &etcdUpdater{instance, instanceDir, instanceFile, nil, nil, nil}
}

// NewDeleter returns an initialized Deleter
func (engine *Engine) NewDeleter(instance *pb.Instance, instancesDir string) state.Deleter {
	instanceDir, instanceFile := helpers.GetInstanceDirFile(instancesDir, instance.Name)
	return &etcdDeleter{instance, instanceDir, instanceFile, nil, nil, nil}
}

type etcdGetter struct {
	*pb.GetOptions
	*pb.InstanceList
	instancesDir string
}

func (getter *etcdGetter) Run() error {
	list := getter.InstanceList

	// setup client
	clientV3, err := clientv3.New(clientv3.Config{Endpoints: endpoints, DialTimeout: dialTimeout})
	if err != nil {
		log.Println(err)
		return err
	}
	helpers.Debugln("Created clientV3")
	defer func() { clientV3.Close(); helpers.Debugln("Closed clientV3") }()
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
		log.Println(err)
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

type etcdTemplatesGetter struct {
	*pb.InstanceList
	templatesDir string
}

func (getter *etcdTemplatesGetter) Run() error {
	// TODO
	return nil
}

type etcdUpdater struct {
	*pb.Instance
	instanceDir  string
	instanceFile string
	clientV3     *clientv3.Client
	session      *concurrency.Session
	mutex        *concurrency.Mutex
}

// Init is expected to do any init related to the state store,
//   as well as write the kustomization.yaml file
func (updater *etcdUpdater) Init() error {
	instanceFile := updater.instanceFile
	instanceKey := getInstanceKey(updater.Instance.Name)

	// setup client
	clientV3, err := clientv3.New(clientv3.Config{Endpoints: endpoints, DialTimeout: dialTimeout})
	if err != nil {
		log.Println(err)
		return err
	}
	helpers.Debugln("Created clientV3")
	updater.clientV3 = clientV3
	kvClient := clientv3.NewKV(clientV3)

	// take out an etcd lock
	session, mutex, err := getSessionAndMutex(clientV3, instanceKey)
	if err != nil {
		log.Println(err)
		clientV3.Close()
		helpers.Debugln("Closed clientV3")
		return err
	}
	updater.session, updater.mutex = session, mutex

	// ensure OldInstance passed in request equals current Instance in etcd
	if err := oldInstanceEqualsCurrentInstanceIfSet(kvClient, instanceKey, updater.Instance.GetOldInstance()); err != nil {
		return updater.cleanUp(err)
	}

	// mkdir and write file
	if err := helpers.MkdirFile(instanceFile, updater.Instance.KustomizationYaml); err != nil {
		return updater.cleanUp(err)
	}

	return nil
}

// Cancel is expected to clean up any mess, and remove the kustomization.yaml file/dir
func (updater *etcdUpdater) Cancel(err error) error {
	return updater.cleanUp(err)
}

// Commit is expected to add/update the Instance in the state store
func (updater *etcdUpdater) Commit() (erR error) {
	instanceKey := getInstanceKey(updater.Instance.Name)
	defer func() { erR = updater.cleanUp(erR) }()

	// at this point, OldInstance matches if present, we have a lock, and
	//   it doesn't matter if the key is pre-existing or not, so just put
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	_, err := updater.clientV3.Put(ctx, instanceKey, updater.Instance.KustomizationYaml)
	cancel()
	if err != nil {
		log.Println(err)
		return err
	}

	return nil

}

func (updater *etcdUpdater) cleanUp(err error) error {
	return cleanUp(updater.mutex, updater.session, updater.clientV3, err)
}

type etcdDeleter struct {
	*pb.Instance
	instanceDir  string
	instanceFile string
	clientV3     *clientv3.Client
	session      *concurrency.Session
	mutex        *concurrency.Mutex
}

// Init is expected to do any init related to the state store,
//   as well as write the kustomization.yaml file
func (deleter *etcdDeleter) Init() error {
	instanceFile := deleter.instanceFile
	instanceKey := getInstanceKey(deleter.Instance.Name)

	// setup client
	clientV3, err := clientv3.New(clientv3.Config{Endpoints: endpoints, DialTimeout: dialTimeout})
	if err != nil {
		log.Println(err)
		return err
	}
	helpers.Debugln("Created clientV3")
	deleter.clientV3 = clientV3
	kvClient := clientv3.NewKV(clientV3)

	// ensure Instance exists in etcd
	txnResponse, err := kvClient.Txn(context.Background()).
		If(clientv3util.KeyExists(instanceKey)).
		Commit()
	if err != nil {
		log.Println(err)
		clientV3.Close()
		helpers.Debugln("Closed clientV3")
		return err
	}
	if !txnResponse.Succeeded { // if !key exists
		clientV3.Close()
		helpers.Debugln("Closed clientV3")
		return fmt.Errorf("No etcd key found for %s", deleter.Instance.Name)
	}

	// take out an etcd lock
	session, mutex, err := getSessionAndMutex(clientV3, instanceKey)
	if err != nil {
		log.Println(err)
		clientV3.Close()
		helpers.Debugln("Closed clientV3")
		return err
	}
	deleter.session, deleter.mutex = session, mutex

	// ensure passed OldInstance equals current Instance in etcd
	if err := oldInstanceEqualsCurrentInstanceIfSet(kvClient, instanceKey, deleter.Instance.GetOldInstance()); err != nil {
		return deleter.cleanUp(err)
	}

	// load kustomizationYaml from etcd
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	response, err := kvClient.Get(ctx, instanceKey)
	cancel()
	if err != nil {
		log.Println(err)
		return deleter.cleanUp(err)
	}
	if len(response.Kvs) != 1 {
		err = fmt.Errorf("Expected 1 key-value set, got %d", len(response.Kvs))
		log.Println(err)
		return err
	}
	deleter.Instance.KustomizationYaml = string(response.Kvs[0].Value)

	// mkdir and write file
	if err := helpers.MkdirFile(instanceFile, deleter.Instance.KustomizationYaml); err != nil {
		return deleter.cleanUp(err)
	}

	return nil
}

// Cancel is expected to clean up any mess, and re-add the kustomization.yaml file/dir
func (deleter *etcdDeleter) Cancel(err error) error {
	return deleter.cleanUp(err)
}

// Commit is expected to delete the Instance from the state store
func (deleter *etcdDeleter) Commit() (erR error) {
	instanceKey := getInstanceKey(deleter.Instance.Name)
	defer func() { erR = deleter.cleanUp(erR) }()

	// at this point, OldInstance matches if present, we have a lock, and
	//   the key exists, so just delete
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	_, err := deleter.clientV3.Delete(ctx, instanceKey, clientv3.WithPrefix())
	cancel()
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (deleter *etcdDeleter) cleanUp(err error) error {
	return cleanUp(deleter.mutex, deleter.session, deleter.clientV3, err)
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
		log.Println(err)
		return &concurrency.Session{}, &concurrency.Mutex{}, err
	}
	helpers.Debugln("Created Concurrency session")

	mutex := concurrency.NewMutex(session, instanceKey)
	if err = mutex.Lock(context.TODO()); err != nil {
		log.Println(err)
		session.Close()
		helpers.Debugln("Closed Concurrency session")
		return session, &concurrency.Mutex{}, err
	}
	helpers.Debugln("Obtained lock for", instanceKey)

	return session, mutex, nil
}

func cleanUp(mutex *concurrency.Mutex, session *concurrency.Session, clientV3 *clientv3.Client, err error) error {
	key := mutex.Key()
	if innerError := mutex.Unlock(context.TODO()); innerError != nil {
		log.Println(err)
		if err != nil {
			session.Close()
			clientV3.Close()
			helpers.Debugln("Closed Concurrency Session and clientV3")
			return fmt.Errorf("error releasing lock after another error: %s\nOriginal Error:\n%s", innerError.Error(), fmt.Sprintf("%s\n", err.Error()))
		}
		session.Close()
		clientV3.Close()
		helpers.Debugln("Closed Concurrency Session and clientV3")
		return fmt.Errorf("error releasing lock: %s", fmt.Sprintf("%s\n", innerError.Error()))
	}
	session.Close()
	clientV3.Close()
	helpers.Debugf("Released lock for %s, closed Concurrency Session and clientV3\n", key)

	return err
}

func oldInstanceEqualsCurrentInstanceIfSet(kvClient clientv3.KV, instanceKey string, oldInstance *pb.Instance) error {
	if oldInstance != nil {
		txnResponse, err := kvClient.Txn(context.Background()).
			If(clientv3util.KeyExists(instanceKey)).
			Commit()
		if err != nil {
			log.Println(err)
			return err
		}
		if !txnResponse.Succeeded { // if !key exists
			return state.InstanceNoExistError
		}
		if txnResponse.Succeeded { // if key exists
			// check for modified since read
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			response, err := kvClient.Get(ctx, instanceKey)
			cancel()
			if err != nil {
				log.Println(err)
				return err
			}
			if len(response.Kvs) != 1 {
				err = fmt.Errorf("Expected 1 key-value set, got %d", len(response.Kvs))
				log.Println(err)
				return err
			}
			currentValue := response.Kvs[0].Value
			if string(currentValue) != oldInstance.KustomizationYaml {
				return state.OldInstanceDiffersError
			}
		}
	}

	return nil
}
