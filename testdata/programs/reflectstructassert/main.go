package main

import (
	"example.com/reflectstructassert/z_consumer"
	"example.com/reflectstructassert/a_producer"
	"fmt"
	"reflect"
)

func main() {
	consumer.Use(producer.Value.Interface())
	consumer.UseOnly(producer.OnlyConsumer.Interface())
	p := producer.Value.Addr().Interface().(*struct{ X int })
	p.X = 8
	fmt.Println(producer.Value.Field(0).Int(), p == producer.Value.Addr().Interface().(*struct{ X int }))
	x := producer.Value.Interface().(struct{ X int })
	x.X = 9
	fmt.Println(producer.Value.Field(0).Int(), x.X)
	s := reflect.MakeSlice(reflect.SliceOf(producer.Type), 2, 2)
	s.Index(0).Set(producer.Value)
	slice := s.Interface().([]struct{ X int })
	slice[0].X = 10
	fmt.Println(s.Index(0).Field(0).Int(), &slice[0] == s.Index(0).Addr().Interface().(*struct{ X int }))
	a := reflect.New(reflect.ArrayOf(2, producer.Type)).Elem()
	a.Index(0).Set(producer.Value)
	array := a.Addr().Interface().(*[2]struct{ X int })
	array[0].X = 11
	fmt.Println(a.Index(0).Field(0).Int())
	m := reflect.MakeMap(reflect.MapOf(producer.Type, producer.Type))
	m.SetMapIndex(producer.Value, producer.Value)
	mp := m.Interface().(map[struct{ X int }]struct{ X int })
	fmt.Println(mp[struct{ X int }{8}].X)
	c := reflect.MakeChan(reflect.ChanOf(reflect.BothDir, producer.Type), 1)
	channel := c.Interface().(chan struct{ X int })
	channel <- struct{ X int }{12}
	v, ok := c.Recv()
	fmt.Println(v.Field(0).Int(), ok)
	st := reflect.StructOf([]reflect.StructField{{Name: "Y", Type: reflect.TypeOf(0)}})
	sv := reflect.New(st).Elem()
	sv.Field(0).SetInt(14)
	switch y := sv.Interface().(type) {
	case struct{ Y int }:
		fmt.Println(y.Y)
	default:
		panic("type switch")
	}
	tagged := reflect.StructOf([]reflect.StructField{{Name: "X", Type: reflect.TypeOf(0), Tag: `json:"x"`}})
	tv := reflect.New(tagged).Elem()
	tv.Field(0).SetInt(13)
	fmt.Println(tv.Interface().(struct {
		X int `json:"x"`
	}).X)
}
