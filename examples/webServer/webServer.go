package main

import (
	pb "github.com/snarlysodboxer/hambone/generated"
	"google.golang.org/grpc"

	"context"
	"errors"
	"flag"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"
)

var (
	templates      = template.Must(template.ParseFiles("new.html", "edit.html", "view.html", "viewall.html"))
	serverAddress  = flag.String("server_address", "127.0.0.1:50051", "Where to reach the hambone server")
	grpcConnection = &grpc.ClientConn{}
)

func saveInstance(instance *pb.Instance) error {
	client := pb.NewInstancesClient(grpcConnection)
	instance, err := client.Apply(context.Background(), instance)
	if err != nil {
		return err
	}

	return nil
}

func loadInstance(name string) (*pb.Instance, error) {
	client := pb.NewInstancesClient(grpcConnection)
	getOptions := &pb.GetOptions{Name: name}
	instance := &pb.Instance{}
	instanceList, err := client.Get(context.Background(), getOptions)
	if err != nil {
		return instance, err
	}
	if instanceList == nil || len(instanceList.Instances) == 0 {
		return instance, errors.New("Not Found")
	}
	instance = instanceList.Instances[0]

	return instance, nil
}

func loadTemplate(name string) (*pb.Instance, error) {
	client := pb.NewInstancesClient(grpcConnection)
	instance := &pb.Instance{}
	templateList, err := client.GetTemplates(context.Background(), &pb.Empty{})
	if err != nil {
		return instance, err
	}
	for _, template := range templateList.Instances {
		if template.Name == name {
			instance = template
			break
		}
	}

	return instance, nil
}

func loadInstances() (*pb.InstanceList, error) {
	client := pb.NewInstancesClient(grpcConnection)
	getOptions := &pb.GetOptions{}
	instanceList, err := client.Get(context.Background(), getOptions)
	if err != nil {
		return instanceList, err
	}

	return instanceList, nil
}

func viewAllHandler(w http.ResponseWriter, r *http.Request) {
	instanceList, err := loadInstances()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = templates.ExecuteTemplate(w, "viewall.html", instanceList)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, name string) {
	instance, err := loadInstance(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	executeTemplate(w, "view", instance)
}

func editHandler(w http.ResponseWriter, r *http.Request, name string) {
	instance, err := loadInstance(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	executeTemplate(w, "edit", instance)
}

func newHandler(w http.ResponseWriter, r *http.Request, name string) {
	template, err := loadTemplate(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	executeTemplate(w, "new", template)
}

func saveHandler(w http.ResponseWriter, r *http.Request, name string) {
	kustomizationYaml := r.FormValue("kustomizationYaml")
	kustomizationYaml = strings.ReplaceAll(kustomizationYaml, "\r", "")
	oldKustomizationYaml := r.FormValue("oldKustomizationYaml")
	oldKustomizationYaml = strings.ReplaceAll(oldKustomizationYaml, "\r", "")
	instance := &pb.Instance{
		Name:              name,
		KustomizationYaml: kustomizationYaml,
		OldInstance: &pb.Instance{Name: name,
			KustomizationYaml: oldKustomizationYaml,
		},
	}
	err := saveInstance(instance)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/instance/view/"+name, http.StatusFound)
}

func saveNewHandler(w http.ResponseWriter, r *http.Request) {
	kustomizationYaml := r.FormValue("kustomizationYaml")
	kustomizationYaml = strings.ReplaceAll(kustomizationYaml, "\r", "")
	name := r.FormValue("name")
	instance := &pb.Instance{
		Name:              name,
		KustomizationYaml: kustomizationYaml,
	}
	err := saveInstance(instance)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/instance/view/"+name, http.StatusFound)
}

func executeTemplate(w http.ResponseWriter, tmpl string, instance *pb.Instance) {
	err := templates.ExecuteTemplate(w, tmpl+".html", instance)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func validationHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	paths := regexp.MustCompile(`^/instance/(new|edit|save|view)/([a-zA-Z0-9\-]+)$`)
	return func(w http.ResponseWriter, r *http.Request) {
		m := paths.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

func main() {
	flag.Parse()

	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	}
	err := errors.New("")
	grpcConnection, err = grpc.Dial(*serverAddress, opts...)
	if err != nil {
		panic(err)
	}
	defer grpcConnection.Close()

	http.HandleFunc("/instance/", viewAllHandler)
	http.HandleFunc("/instance/new/", validationHandler(newHandler))
	http.HandleFunc("/instance/view/", validationHandler(viewHandler))
	http.HandleFunc("/instance/edit/", validationHandler(editHandler))
	http.HandleFunc("/instance/save/", validationHandler(saveHandler))
	http.HandleFunc("/instance/savenew/", saveNewHandler)

	log.Println("Listening on localhost:8080")
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}
