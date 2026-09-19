package main

import (
	"encoding/json"
	"fmt"
	"reflect"

	"charm.land/fantasy/jsonrepair"
	"charm.land/fantasy/schema"
)

func main() {
	for _, input := range []string{
		`{"name":"Ada","values":[1,2,3]}`,
		`{'name': 'Ada', 'values': [1,2,3,]}`,
		`{"name":"Ada`,
		`[true, false, null]`,
	} {
		output, err := jsonrepair.RepairJSON(input)
		if err != nil {
			panic(err)
		}
		fmt.Println(output)
	}
	model := schema.Generate(reflect.TypeOf(struct {
		Name string `json:"name" description:"Display name"`
		Age  int    `json:"age"`
	}{}))
	encoded, err := json.Marshal(schema.ToParameters(model))
	if err != nil {
		panic(err)
	}
	fmt.Println(string(encoded))
}
