package etcd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"

	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/pkg/helpers"
	"github.com/snarlysodboxer/hambone/pkg/state"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/clientv3/clientv3util"
	"go.etcd.io/etcd/clientv3/concurrency"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	dialTimeout       = 5 * time.Second
	requestTimeout    = 5 * time.Second
	lockTimeout       = 30 * time.Second
	instanceKeyPrefix = "hambone_instance_"
)

var (
	endpoints = []string{}
)

// Engine fulfills the StateStore interface
type Engine struct {
	WorkingDir      string
	EndpointsString string
}

// Init does setup
func (engine *Engine) Init() error {
	err := os.Chdir(engine.WorkingDir)
	if err != nil {
		return status.Errorf(codes.Unknown, "error changing directory: %s", err.Error())
	}

	// parse and set endpoints
	endpoints = strings.Split(engine.EndpointsString, ",")

	return nil
}

// NewGetter returns an initialized Getter
func (engine *Engine) NewGetter(options *pb.GetOptions, list *pb.InstanceList, instancesDir string) state.Getter {
	return &etcdGetter{options, list, instancesDir, nil}
}

// NewTemplatesGetter returns an initialized TemplatesGetter
func (engine *Engine) NewTemplatesGetter(list *pb.InstanceList, templatesDir string) state.TemplatesGetter {
	return &etcdTemplatesGetter{list, templatesDir, nil}
}

// NewUpdater returns an initialized Updater
func (engine *Engine) NewUpdater(instance *pb.Instance, instancesDir string) state.Updater {
	instanceDir, instanceFile := helpers.GetInstanceDirFile(instancesDir, instance.Name)
	return &etcdUpdater{instance, instanceDir, instanceFile, nil, nil}
}

// NewDeleter returns an initialized Deleter
func (engine *Engine) NewDeleter(instance *pb.Instance, instancesDir string) state.Deleter {
	instanceDir, instanceFile := helpers.GetInstanceDirFile(instancesDir, instance.Name)
	return &etcdDeleter{instance, instanceDir, instanceFile, nil, nil}
}

type etcdGetter struct {
	*pb.GetOptions
	*pb.InstanceList
	instancesDir string
	cleanupFuncs []func() error
}

func (getter *etcdGetter) Run() error {
	list := getter.InstanceList

	// setup client
	clientV3, err := clientv3.New(clientv3.Config{Endpoints: endpoints, DialTimeout: dialTimeout})
	if err != nil {
		return err
	}
	helpers.Debugln("Created clientV3")
	getter.cleanupFuncs = append(getter.cleanupFuncs, func() error {
		err = clientV3.Close()
		if err != nil {
			return err
		}
		helpers.Debugln("Closed clientV3")
		return nil
	})
	kvClient := clientv3.NewKV(clientV3)

	// get key-values
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	response := &clientv3.GetResponse{}
	if getter.GetOptions.GetName() != "" {
		response, err = kvClient.Get(ctx, getInstanceKey(getter.GetOptions.GetName()))
	} else {
		response, err = kvClient.Get(ctx, instanceKeyPrefix, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))

	}
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

func (getter *etcdGetter) RunCleanupFuncs() error {
	err := runCleanupFuncs(getter.cleanupFuncs)
	getter.cleanupFuncs = nil
	return err
}

type etcdTemplatesGetter struct {
	*pb.InstanceList
	templatesDir string
	cleanupFuncs []func() error
}

func (getter *etcdTemplatesGetter) Run() error {
	// TODO
	// getter.cleanupFuncs = append(getter.cleanupFuncs, func() error {
	//     err = clientV3.Close()
	//     if err != nil {
	//         return err
	//     }
	//     helpers.Debugln("Closed clientV3")
	//     return nil
	// })
	return nil
}

func (getter *etcdTemplatesGetter) RunCleanupFuncs() error {
	err := runCleanupFuncs(getter.cleanupFuncs)
	getter.cleanupFuncs = nil
	return err
}

type etcdUpdater struct {
	*pb.Instance
	instanceDir  string
	instanceFile string
	clientV3     *clientv3.Client
	cleanupFuncs []func() error
}

// Init is expected to do any init related to the state store,
//   as well as write the kustomization.yaml file
func (updater *etcdUpdater) Init() error {
	instanceFile := updater.instanceFile
	instanceKey := getInstanceKey(updater.Instance.Name)

	// setup client
	clientV3, err := clientv3.New(clientv3.Config{Endpoints: endpoints, DialTimeout: dialTimeout})
	updater.clientV3 = clientV3
	if err != nil {
		return err
	}
	helpers.Debugln("Created updater clientV3")
	updater.cleanupFuncs = append(updater.cleanupFuncs, func() error {
		err = clientV3.Close()
		if err != nil {
			return err
		}
		helpers.Debugln("Closed updater clientV3")
		return nil
	})
	kvClient := clientv3.NewKV(clientV3)

	// take out an etcd lock
	cancel, session, mutex, err := GetSessionAndMutex(clientV3, instanceKey, "updater")
	if err != nil {
		return err
	}
	updater.cleanupFuncs = append(updater.cleanupFuncs, func() error {
		cancel()
		helpers.Debugln("Ran updater CancelFunc")
		return nil
	})
	updater.cleanupFuncs = append(updater.cleanupFuncs, func() error {
		if err = session.Close(); err != nil {
			return err
		}
		helpers.Debugln("Closed updater session")
		return nil
	})
	updater.cleanupFuncs = append(updater.cleanupFuncs, func() error {
		key := mutex.Key()
		if err = mutex.Unlock(context.TODO()); err != nil {
			return err
		}
		helpers.Debugf("Released updater lock for %s", key)
		return nil
	})

	// ensure OldInstance passed in request equals current Instance in etcd
	if err := oldInstanceEqualsCurrentInstanceIfSet(kvClient, instanceKey, updater.Instance.GetOldInstance()); err != nil {
		return err
	}

	// mkdir and write file
	if err := helpers.MkdirFile(instanceFile, updater.Instance.KustomizationYaml); err != nil {
		return err
	}
	for _, file := range updater.Instance.Files {
		if err = helpers.MkdirFile(filepath.Join(updater.instanceDir, file.Name), file.Contents); err != nil {
			return err
		}
	}

	return nil
}

// Cancel is expected to clean up any mess, and remove the kustomization.yaml file/dir
func (updater *etcdUpdater) Cancel(err error) error {
	// TODO
	return err
}

// Commit is expected to add/update the Instance in the state store
func (updater *etcdUpdater) Commit() (erR error) {
	instanceKey := getInstanceKey(updater.Instance.Name)

	// at this point, OldInstance matches if present, we have a lock, and
	//   it doesn't matter if the key is pre-existing or not, so just put
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	_, err := updater.clientV3.Put(ctx, instanceKey, updater.Instance.KustomizationYaml)
	if err != nil {
		return err
	}
	for _, file := range updater.Instance.Files {
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()
		fileKey := getFileKey(instanceKey, file.Name)
		_, err := updater.clientV3.Put(ctx, fileKey, file.Contents)
		if err != nil {
			return err
		}
	}

	return nil

}

func (updater *etcdUpdater) RunCleanupFuncs() error {
	err := runCleanupFuncs(updater.cleanupFuncs)
	updater.cleanupFuncs = nil
	return err
}

type etcdDeleter struct {
	*pb.Instance
	instanceDir  string
	instanceFile string
	clientV3     *clientv3.Client
	cleanupFuncs []func() error
}

// Init is expected to do any init related to the state store,
//   as well as write the kustomization.yaml file
func (deleter *etcdDeleter) Init() error {
	instanceFile := deleter.instanceFile
	instanceKey := getInstanceKey(deleter.Instance.Name)

	// setup client
	clientV3, err := clientv3.New(clientv3.Config{Endpoints: endpoints, DialTimeout: dialTimeout})
	if err != nil {
		return err
	}
	helpers.Debugln("Created deleter clientV3")
	deleter.cleanupFuncs = append(deleter.cleanupFuncs, func() error {
		err = clientV3.Close()
		if err != nil {
			return err
		}
		helpers.Debugln("Closed deleter clientV3")
		return nil
	})
	deleter.clientV3 = clientV3
	kvClient := clientv3.NewKV(clientV3)

	// ensure Instance exists in etcd
	txnResponse, err := kvClient.Txn(context.Background()).
		If(clientv3util.KeyExists(instanceKey)).
		Commit()
	if err != nil {
		return err
	}
	if !txnResponse.Succeeded { // if !key exists
		return fmt.Errorf("No etcd key found for %s", deleter.Instance.Name)
	}

	// take out an etcd lock
	cancel, session, mutex, err := GetSessionAndMutex(clientV3, instanceKey, "deleter")
	if err != nil {
		return err
	}
	deleter.cleanupFuncs = append(deleter.cleanupFuncs, func() error {
		cancel()
		helpers.Debugln("Ran deleter CancelFunc")
		return nil
	})
	deleter.cleanupFuncs = append(deleter.cleanupFuncs, func() error {
		if err = session.Close(); err != nil {
			return err
		}
		helpers.Debugln("Closed deleter session")
		return nil
	})
	deleter.cleanupFuncs = append(deleter.cleanupFuncs, func() error {
		key := mutex.Key()
		if err = mutex.Unlock(context.TODO()); err != nil {
			return err
		}
		helpers.Debugf("Released deleter lock for %s", key)
		return nil
	})

	// ensure passed OldInstance equals current Instance in etcd
	if err := oldInstanceEqualsCurrentInstanceIfSet(kvClient, instanceKey, deleter.Instance.GetOldInstance()); err != nil {
		return err
	}

	// load kustomizationYaml from etcd
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	response, err := kvClient.Get(ctx, instanceKey)
	if err != nil {
		return err
	}
	if len(response.Kvs) != 1 {
		err = fmt.Errorf("Expected 1 key-value set, got %d", len(response.Kvs))
		return err
	}
	deleter.Instance.KustomizationYaml = string(response.Kvs[0].Value)

	// mkdir and write file for kustomize build | kubectl delete
	if err := helpers.MkdirFile(instanceFile, deleter.Instance.KustomizationYaml); err != nil {
		return err
	}
	for _, file := range deleter.Instance.Files {
		if err = helpers.MkdirFile(filepath.Join(deleter.instanceDir, file.Name), file.Contents); err != nil {
			return err
		}
	}

	return nil
}

// Cancel is expected to clean up any mess, and re-add the kustomization.yaml file/dir
func (deleter *etcdDeleter) Cancel(err error) error {
	return err
}

// Commit is expected to delete the Instance from the state store
func (deleter *etcdDeleter) Commit() (erR error) {
	instanceKey := getInstanceKey(deleter.Instance.Name)

	// at this point, OldInstance matches if present, we have a lock, and
	//   the key exists, so just delete
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	_, err := deleter.clientV3.Delete(ctx, instanceKey, clientv3.WithPrefix())
	if err != nil {
		return err
	}
	return nil
}

func (deleter *etcdDeleter) RunCleanupFuncs() error {
	err := runCleanupFuncs(deleter.cleanupFuncs)
	deleter.cleanupFuncs = nil
	return err
}

func runCleanupFuncs(funcs []func() error) error {
	// reverse slice order
	for i := len(funcs)/2 - 1; i >= 0; i-- {
		opp := len(funcs) - 1 - i
		funcs[i], funcs[opp] = funcs[opp], funcs[i]
	}

	errorString := "Cleanup Errors: "
	isError := false
	for _, fn := range funcs {
		err := fn()
		if err != nil {
			isError = true
			name := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
			errorString += fmt.Sprintf("\n%s: %s\n", name, err.Error())
		}
	}
	if isError {
		return fmt.Errorf(errorString)
	}

	return nil
}

func getInstanceKey(name string) string {
	return fmt.Sprintf("%s%s", instanceKeyPrefix, name)
}

func getFileKey(instanceKey, name string) string {
	return fmt.Sprintf("%s/%s", instanceKey, name)
}

func stripInstanceKeyPrefix(name string) string {
	return name[len(instanceKeyPrefix):]
}

// GetSessionAndMutex gets an etcd lock
func GetSessionAndMutex(clientV3 *clientv3.Client, key, caller string) (context.CancelFunc, *concurrency.Session, *concurrency.Mutex, error) {
	// do not allow lock key to collide with a storage key!
	key = fmt.Sprintf("%s-lock", key)
	ctx, cancel := context.WithTimeout(context.Background(), lockTimeout)
	session, err := concurrency.NewSession(clientV3, concurrency.WithContext(ctx))
	if err != nil {
		return cancel, &concurrency.Session{}, &concurrency.Mutex{}, err
	}
	helpers.Debugf("Created %s Concurrency session", caller)

	mutex := concurrency.NewMutex(session, key)
	ctx, cncl := context.WithTimeout(context.Background(), lockTimeout)
	defer cncl()
	if err = mutex.Lock(ctx); err != nil {
		session.Close()
		helpers.Debugf("Closed %s Concurrency session because of error with lock", caller)
		return cancel, session, &concurrency.Mutex{}, err
	}
	helpers.Debugf("Obtained %s lock for %s", caller, mutex.Key())

	return cancel, session, mutex, nil
}

func oldInstanceEqualsCurrentInstanceIfSet(kvClient clientv3.KV, instanceKey string, oldInstance *pb.Instance) error {
	if oldInstance != nil {
		txnResponse, err := kvClient.Txn(context.Background()).
			If(clientv3util.KeyExists(instanceKey)).
			Commit()
		if err != nil {
			return err
		}
		if !txnResponse.Succeeded { // if !key exists
			return state.InstanceNoExistError
		}
		if txnResponse.Succeeded { // if key exists
			// check for modified since read
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			response, err := kvClient.Get(ctx, instanceKey)
			defer cancel()
			if err != nil {
				return err
			}
			if len(response.Kvs) != 1 {
				err = fmt.Errorf("Expected 1 key-value set, got %d", len(response.Kvs))
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
