package main

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
)

func trees[T comparable](value T) {
	type tree struct {
		Value    T
		Children []tree
	}
	child := tree{Value: value}
	root := tree{Value: value, Children: []tree{child}}
	fmt.Println(root.Value == value, root.Children[0].Value == value)
	fmt.Println(slices.Clone(root.Children)[0].Value == value)
	copy := root
	copy.Children = append(copy.Children, child)
	fmt.Println(len(root.Children), len(copy.Children))
	var holder any = root
	restored := holder.(tree)
	fmt.Println(reflect.DeepEqual(root, restored))
	pointer := reflect.New(reflect.TypeFor[tree]())
	pointer.Elem().Set(reflect.ValueOf(root))
	other := pointer.Interface().(*tree)
	fmt.Println(other.Value == value, other.Children[0].Value == value)
	pointer.Elem().Field(1).Index(0).Field(0).Set(reflect.ValueOf(value))
	type view tree
	converted := (*view)(&root)
	converted.Children = append(converted.Children, child)
	fmt.Println(len(root.Children))
	fmt.Println(slices.Clone(root.Children)[0].Value == value)
	unnamed := struct {
		Value    T
		Children []tree
	}(root)
	fmt.Println(unnamed.Children[0].Value == value)
	list := []tree{root}
	array := [1]tree{root}
	table := map[int]tree{0: root}
	ch := make(chan tree, 1)
	ch <- root
	fmt.Println(list[0].Value == value, array[0].Value == value, table[0].Value == value, (<-ch).Value == value)
	backing := [1]tree{{Value: value}}
	root = tree{Value: value, Children: backing[:]}
	fmt.Println(slices.Clone(root.Children)[0].Value == value)
	children := &root.Children
	fmt.Println(slices.Clone(*children)[0].Value == value)
	shape := reflect.StructOf([]reflect.StructField{
		{Name: "Value", Type: reflect.TypeFor[T]()},
		{Name: "Children", Type: reflect.SliceOf(reflect.TypeFor[tree]())},
	})
	dynamic := reflect.New(shape).Elem()
	dynamic.Field(0).Set(reflect.ValueOf(value))
	dynamic.Field(1).Set(reflect.ValueOf(root.Children))
	rebuilt := dynamic.Convert(reflect.TypeFor[tree]()).Interface().(tree)
	fmt.Println(rebuilt.Value == value, rebuilt.Children[0].Value == value)
	type mapTree struct {
		Value    T
		Children map[string]mapTree
	}
	mapped := mapTree{Value: value, Children: map[string]mapTree{"child": {Value: value}}}
	fmt.Println(maps.Clone(mapped.Children)["child"].Value == value)
	type chanTree struct {
		Value    T
		Children chan chanTree
	}
	queued := chanTree{Value: value, Children: make(chan chanTree, 1)}
	queued.Children <- chanTree{Value: value}
	fmt.Println((<-queued.Children).Value == value)
	type funcTree struct {
		Value T
		Child func() funcTree
	}
	called := funcTree{Value: value, Child: func() funcTree { return funcTree{Value: value} }}
	fmt.Println(called.Child().Value == value)
}

func main() {
	trees(7)
	trees("value")
	type mapNode struct {
		Value    int
		Children map[string]mapNode
	}
	m := mapNode{1, map[string]mapNode{"child": {Value: 2}}}
	fmt.Println(m.Children["child"].Value)
	found, ok := m.Children["child"]
	missing, absent := m.Children["missing"]
	fmt.Println(found.Value, ok, missing.Value, absent)
	for key, item := range m.Children {
		fmt.Println(key, item.Value)
	}
	reflectedMap := reflect.MakeMap(reflect.TypeOf(m.Children))
	reflectedMap.SetMapIndex(reflect.ValueOf("copy"), reflect.ValueOf(found))
	m.Children = reflectedMap.Interface().(map[string]mapNode)
	for key, item := range m.Children {
		fmt.Println(key, item.Value)
	}
	type chanNode struct {
		Value    int
		Children chan chanNode
	}
	c := chanNode{1, make(chan chanNode, 1)}
	c.Children <- chanNode{Value: 2}
	received, open := <-c.Children
	fmt.Println(received.Value, open)
	c.Children <- chanNode{Value: 3}
	select {
	case item := <-c.Children:
		fmt.Println(item.Value)
	default:
		panic("missing item")
	}
	close(c.Children)
	closed, open := <-c.Children
	fmt.Println(closed.Value, open)
	reflectedChan := reflect.MakeChan(reflect.TypeOf(c.Children), 1)
	reflectedChan.Send(reflect.ValueOf(chanNode{Value: 4}))
	c.Children = reflectedChan.Interface().(chan chanNode)
	fmt.Println((<-c.Children).Value)
	type funcNode struct {
		Value int
		Child func() funcNode
	}
	f := funcNode{Value: 1}
	f.Child = func() funcNode { return funcNode{Value: 2} }
	fmt.Println(f.Child().Value)
	f.Child = reflect.MakeFunc(reflect.TypeOf(f.Child), func([]reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(funcNode{Value: 3})}
	}).Interface().(func() funcNode)
	fmt.Println(f.Child().Value)
	type nested struct {
		Value    int
		Children []struct{ Item nested }
	}
	n := nested{Value: 1, Children: []struct{ Item nested }{{Item: nested{Value: 2}}}}
	fmt.Println(n.Children[0].Item.Value)
	type arrayNode struct {
		Value    int
		Children [1][]arrayNode
	}
	a := arrayNode{Value: 1, Children: [1][]arrayNode{{{Value: 2}}}}
	fmt.Println(a.Children[0][0].Value)
	type recursiveSlice []recursiveSlice
	type sliceHolder struct{ Children recursiveSlice }
	h := sliceHolder{Children: recursiveSlice{nil}}
	fmt.Println(len(h.Children), h.Children[0] == nil, reflect.ValueOf(h).Field(0).Len())
	type recursiveMap map[string]recursiveMap
	type mapHolder struct{ Children recursiveMap }
	mh := mapHolder{Children: recursiveMap{"child": nil}}
	fmt.Println(len(mh.Children), mh.Children["child"] == nil, reflect.ValueOf(mh).Field(0).Len())
	entry, present := mh.Children["child"]
	fmt.Println(entry == nil, present)
	missingRecursive, hasRecursive := mh.Children["missing"]
	fmt.Println(missingRecursive == nil, len(missingRecursive), hasRecursive)
	deeper := missingRecursive["deeper"]
	fmt.Println(deeper == nil, len(deeper))
	delete(mh.Children, "child")
	fmt.Println(len(mh.Children))
	mh.Children["only"] = nil
	for key, child := range mh.Children {
		fmt.Println(key, child == nil)
	}
	clear(mh.Children)
	fmt.Println(len(mh.Children))
	type recursiveChan chan recursiveChan
	queue := make(recursiveChan, 1)
	queue <- nil
	fmt.Println(len(queue), cap(queue))
	select {
	case child, open := <-queue:
		fmt.Println(child == nil, open)
	default:
		panic("missing child")
	}
	close(queue)
	child, open := <-queue
	fmt.Println(child == nil, open)
	h.Children = append(h.Children, h.Children[:0]...)
	target := make(recursiveSlice, 1)
	fmt.Println(copy(target, h.Children), target[0] == nil)
	clear(target)
	fmt.Println(target[0] == nil)
}
