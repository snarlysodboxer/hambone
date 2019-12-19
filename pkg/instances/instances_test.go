package instances

import (
	pb "github.com/snarlysodboxer/hambone/generated"

	"testing"
)

func TestNamesEquate(t *testing.T) {
	// Differing
	instance := &Instance{Instance: &pb.Instance{Name: "asdf", KustomizationYaml: ""}}
	instance.Instance.OldInstance = &pb.Instance{Name: "fdsa", KustomizationYaml: ""}
	err := NamesEquate(instance)
	if err == nil {
		t.Error("Expected an error when Instance Name and OldInstance Name are not the same")
	}

	// Equal
	instance = &Instance{Instance: &pb.Instance{Name: "asdf", KustomizationYaml: ""}}
	instance.Instance.OldInstance = &pb.Instance{Name: "asdf", KustomizationYaml: ""}
	err = NamesEquate(instance)
	if err != nil {
		t.Error("Expected no error when Instance Name and OldInstance Name are the same")
	}
}
