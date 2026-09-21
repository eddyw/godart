package main

import (
	"encoding/asn1"
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

func main() {
	zero := time.Time{}
	date := time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)
	parsed, err := time.Parse(time.RFC3339, "0001-01-01T00:00:00Z")
	if err != nil {
		panic(err)
	}
	located, err := time.ParseInLocation("2006-01-02T15:04:05-0700", "0001-01-01T00:00:00+0000", time.UTC)
	if err != nil {
		panic(err)
	}
	binary, err := zero.MarshalBinary()
	if err != nil {
		panic(err)
	}
	var decoded time.Time
	if err := decoded.UnmarshalBinary(binary); err != nil {
		panic(err)
	}
	text, err := json.Marshal(zero)
	if err != nil {
		panic(err)
	}
	var jsonTime time.Time
	if err := json.Unmarshal(text, &jsonTime); err != nil {
		panic(err)
	}
	for _, value := range []time.Time{zero.UTC(), zero.In(time.UTC), date, parsed, located, decoded, jsonTime} {
		fmt.Println(value == zero, reflect.DeepEqual(value, zero), value.Location() == time.UTC)
	}
	value := zero.Add(time.Hour)
	der, err := asn1.Marshal(value)
	if err != nil {
		panic(err)
	}
	var decodedASN1 time.Time
	if _, err := asn1.Unmarshal(der, &decodedASN1); err != nil {
		panic(err)
	}
	fmt.Println(value == decodedASN1, reflect.DeepEqual(value, decodedASN1))
	fmt.Println(zero.In(time.FixedZone("UTC", 0)) == zero)
	original := time.UTC
	time.UTC = time.FixedZone("replacement", 3600)
	fmt.Println(zero.UTC() == zero, zero.In(time.UTC) == zero)
	time.UTC = original
}
