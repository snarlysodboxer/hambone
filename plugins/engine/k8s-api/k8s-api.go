package main

import (
	"errors"
	"fmt"
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/plugins/render"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/apps/v1beta1"
	"reflect"
	// "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	// "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
	// "k8s.io/kubernetes/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	actionApply  = "apply"
	actionDelete = "delete"
	actionStatus = "status"
)

var K8sEngine K8sAPIEngine

type K8sAPIEngine struct {
	renderer           render.Interface
	ClientSet          kubernetes.Interface
	kubeConfigFilePath string
}

func NewK8sAPIEngine(renderer render.Interface) *K8sAPIEngine {
	return &K8sAPIEngine{renderer, &kubernetes.Clientset{}, ""}
}

func (engine *K8sAPIEngine) Init(renderer render.Interface, kubeConfigFilePath string, inClusterConfig bool) error {
	engine.kubeConfigFilePath = kubeConfigFilePath
	engine.renderer = renderer
	restConfig := &rest.Config{}
	if inClusterConfig {
		config, err := rest.InClusterConfig()
		if err != nil {
			return err
		}
		restConfig = config
	} else {
		config, err := clientcmd.BuildConfigFromFlags("", kubeConfigFilePath)
		if err != nil {
			return err
		}
		restConfig = config
	}
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	engine.ClientSet = clientSet
	return nil
}

// Apply renders and applies
func (engine *K8sAPIEngine) ApplyInstance(instance *pb.Instance) error {
	err := engine.crud(actionApply, instance)
	if err != nil {
		return err
	}
	return nil
}

// Delete renders and deletes
func (engine *K8sAPIEngine) DeleteInstance(instance *pb.Instance) error {
	err := engine.crud(actionDelete, instance)
	if err != nil {
		return err
	}
	return nil
}

// Status returns status information
// func (engine *K8sAPIEngine) StatusInstance(instance *pb.Instance) (*pb.StatusMessage, error) {
//     err := engine.crud(actionStatus, instance)
//     if err != nil {
//         return err
//     }
//     return nil
// }

func (engine *K8sAPIEngine) crud(action string, instance *pb.Instance) error {
	// TODO figure out how to DRY this
	rendereds, err := engine.renderer.Render(instance)
	if err != nil {
		return err
	}
	for _, rendered := range rendereds {
		obj, err := parseYaml(rendered)
		if err != nil {
			return err
		}
		switch object := obj.(type) {
		case *v1beta1.Deployment:
			fmt.Println("get here")
			client := engine.ClientSet.AppsV1beta1().Deployments(apiv1.NamespaceDefault)
			exists := true
			if _, err := client.Get(object.ObjectMeta.Name, metav1.GetOptions{}); err != nil {
				if k8sErrors.IsNotFound(err) {
					exists = false
				} else {
					return err
				}
			}
			switch action {
			case actionApply:
				if exists {
					if _, err := client.Update(object); err != nil {
						return err
					}
					return nil
				}
				if _, err := client.Create(object); err != nil {
					return err
				}
				return nil
			case actionDelete:
				if exists {
					if err := client.Delete(object.ObjectMeta.Name, &metav1.DeleteOptions{}); err != nil {
						return err
					}
					return nil
				}
				return nil
			case actionStatus:
				if !exists {
					return errors.New(fmt.Sprintf("Cannot get status for non-existent %T '%s'", object, object.ObjectMeta.Name))
				}
				// TODO
				return nil
			default:
				return errors.New("Unknown action")
			}
		case *apiv1.Service:
			client := engine.ClientSet.CoreV1().Services(apiv1.NamespaceDefault)
			exists := true
			if _, err := client.Get(object.ObjectMeta.Name, metav1.GetOptions{}); err != nil {
				if k8sErrors.IsNotFound(err) {
					exists = false
				} else {
					return err
				}
			}
			switch action {
			case actionApply:
				if exists {
					if _, err := client.Update(object); err != nil {
						return err
					}
					return nil
				}
				if _, err := client.Create(object); err != nil {
					return err
				}
				return nil
			case actionDelete:
				if exists {
					if err := client.Delete(object.ObjectMeta.Name, &metav1.DeleteOptions{}); err != nil {
						return err
					}
					return nil
				}
				return nil
			case actionStatus:
				if !exists {
					return errors.New(fmt.Sprintf("Cannot get status for non-existent %T '%s'", object, object.ObjectMeta.Name))
				}
				// TODO
				return nil
			default:
				return errors.New("Unknown action")
			}
		default:
			fmt.Println(fmt.Sprintf("Type Path %s\n", reflect.ValueOf(obj).Type()))
			return errors.New(fmt.Sprintf("I don't know the object Type %T\n", obj))
		}
	}
	return nil
}

func parseYaml(yaml string) (runtime.Object, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	object, _, err := decode([]byte(yaml), nil, nil)
	if err != nil {
		return object, err
	}
	return object, nil
}
