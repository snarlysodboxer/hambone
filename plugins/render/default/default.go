package main

import (
	"bytes"
	"encoding/json"
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/plugins/state"
	"text/template"
)

var Renderer DefaultRenderer

type DefaultRenderer struct {
	stateStore state.Interface
}

func (renderer *DefaultRenderer) SetStateStore(stateStore state.Interface) {
	renderer.stateStore = stateStore
}

// Render renders each ValueSet against it's referenced Spec
func (renderer *DefaultRenderer) Render(instance *pb.Instance) ([]string, error) {
	rendereds := []string{}
	for _, valueSet := range instance.ValueSets {
		specGroup, err := renderer.stateStore.ReadSpecGroup(instance.SpecGroupId)
		if err != nil {
			return []string{""}, err
		}
		for _, spec := range specGroup.Specs {
			if spec.Id == valueSet.SpecId {
				var parsedTemplate interface{}
				err := json.Unmarshal([]byte(valueSet.JsonBlob), &parsedTemplate)
				if err != nil {
					return []string{""}, err
				}
				t := template.Must(template.New("specification").Parse(spec.Template))
				buffer := new(bytes.Buffer)
				err = t.Execute(buffer, parsedTemplate)
				if err != nil {
					return []string{""}, err
				}
				rendereds = append(rendereds, buffer.String())
			}
		}
	}

	return rendereds, nil
}
