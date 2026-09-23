package producer

import "reflect"

var Type = reflect.StructOf([]reflect.StructField{{Name: "X", Type: reflect.TypeOf(0)}})
var Value = reflect.New(Type).Elem()

var OnlyConsumerType = reflect.StructOf([]reflect.StructField{{Name: "Z", Type: reflect.TypeOf(0)}})
var OnlyConsumer = reflect.New(OnlyConsumerType).Elem()

func init() {
	Value.Field(0).SetInt(7)
	OnlyConsumer.Field(0).SetInt(15)
}
