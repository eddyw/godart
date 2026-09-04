// The program proves that debug/gosym reads a symbol table and that it rejects
// a PC/line table with a wrong magic value.
package main

import (
	"debug/gosym"
	"fmt"
)

func newSym(value uint64, kind byte, name string) gosym.Sym {
	return gosym.Sym{Value: value, Type: kind, Name: name}
}

func main() {
	main1 := newSym(0x1000, 'T', "main.main")
	helper := newSym(0x1200, 't', "main.(*buffer).grow")
	plain := newSym(0x1400, 'T', "fmt.Println")
	data := newSym(0x2000, 'D', "main.count")

	funcs := []gosym.Func{
		{Entry: 0x1000, End: 0x1200, Sym: &main1},
		{Entry: 0x1200, End: 0x1400, Sym: &helper},
		{Entry: 0x1400, End: 0x1600, Sym: &plain},
	}
	obj := gosym.Obj{Funcs: funcs}
	table := &gosym.Table{
		Syms:  []gosym.Sym{main1, helper, plain, data},
		Funcs: funcs,
		Objs:  []gosym.Obj{obj},
		Files: map[string]*gosym.Obj{"main.go": &obj},
	}

	for _, s := range table.Syms {
		fmt.Printf("%s value=%#x static=%t package=%q receiver=%q base=%q\n",
			s.Name, s.Value, s.Static(), s.PackageName(), s.ReceiverName(), s.BaseName())
	}

	for _, name := range []string{"main.main", "fmt.Println", "main.count", "main.absent"} {
		s := table.LookupSym(name)
		f := table.LookupFunc(name)
		fmt.Printf("lookup %s sym=%t func=%t\n", name, s != nil, f != nil)
	}

	for _, pc := range []uint64{0x0fff, 0x1000, 0x1300, 0x15ff, 0x1600} {
		f := table.PCToFunc(pc)
		name := "none"
		if f != nil {
			name = f.Name
		}
		fmt.Printf("pc %#x -> %s\n", pc, name)
	}

	for _, addr := range []uint64{0x1000, 0x1100, 0x2000, 0x0100} {
		s := table.SymByAddr(addr)
		name := "none"
		if s != nil {
			name = s.Name
		}
		fmt.Printf("addr %#x -> %s\n", addr, name)
	}

	short := gosym.NewLineTable([]byte{0xfb, 0xff, 0xff, 0xff}, 0x1000)
	shortTable, err := gosym.NewTable(nil, short)
	fmt.Println("short table", len(shortTable.Funcs), err)

	wrong := make([]byte, 32)
	wrong[0], wrong[1], wrong[2], wrong[3] = 0xaa, 0xbb, 0xcc, 0xdd
	wrong[6], wrong[7] = 1, 8
	bad := gosym.NewLineTable(wrong, 0x1000)
	badTable, err := gosym.NewTable(nil, bad)
	fmt.Println("wrong magic", len(badTable.Funcs), err)
	fmt.Println("line of a wrong table", bad.PCToLine(0x1000))
	fmt.Println("pc of a wrong table", bad.LineToPC(1, 0x2000))
}
