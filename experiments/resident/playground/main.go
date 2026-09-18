package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func greet(name string) string {
	return "hello, " + name
}

func main() {
	println(greet("world"))

	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)

	fmt.Printf("Now time!: { %v }\n", now.Format(time.RFC3339Nano))

	n := 0
	for i := range 5 {
		n += i * i
	}
	println(n)

	res, err := http.Get("http://127.0.0.1:0/")
	if err != nil {
		panic(err)
	}
	res.Body.Close()

	val := 123
	ptr := &val

	*ptr = 456

	fmt.Println(val, ptr)
	fmt.Printf("%v %v", val, ptr)

	var (
		person = Person{
			Name: "Alice" + " Smith!",
			Age:  33 + 7,
		}
	)
	println(person.Name, person.Age)

	jsonData, err := json.MarshalIndent(person, "", "  ")
	if err != nil {
		println("Error marshaling JSON:", err.Error())
		return
	}
	fmt.Printf("JSON data: %s\n", string(jsonData))

	thing := map[string]string{
		"key": "hello world!",
	}
	fmt.Printf("map is %v\n", thing)
}
