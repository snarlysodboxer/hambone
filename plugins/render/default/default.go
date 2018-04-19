package main

import (
	"encoding/json"
	"fmt"
	pb "github.com/snarlysodboxer/hambone/generated"
	"os"
	"text/template"
)

const letter = `
Hello {{.Name}},
{{.Order}}
Regards,
Mosie
`

var Renderer DefaultRenderer

type DefaultRenderer struct {
}

func (renderer *DefaultRenderer) Render(instance *pb.Instance) ([]string, error) {
	// for _, valueSet := range instance.value_sets {
	// }

	var jsonBlob = []byte(`
    {"Name": "Ruoll",    "Order": 6}
`)

	var parsed interface{}
	err := json.Unmarshal(jsonBlob, &parsed)
	if err != nil {
		fmt.Println("error:", err)
	}
	t := template.Must(template.New("letter").Parse(letter))
	err = t.Execute(os.Stdout, parsed)
	if err != nil {
		fmt.Println("executing template:", err)
	}

	return []string{""}, nil
}
