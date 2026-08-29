package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
	// The package variable flag.Usage holds a function literal that can block.
	// The init routine of the package writes it as a local function. The
	// report names os.Args[0], and that name differs, thus the output holds
	// the count of lines only.
	var usage strings.Builder
	flag.CommandLine.SetOutput(&usage)
	flag.Usage()
	fmt.Println("usage lines:", strings.Count(usage.String(), "\n"))
	col := flag.Int("col", 1, "the column to add")
	name := flag.String("name", "report", "the name of the report")
	flag.Parse()

	r := csv.NewReader(strings.NewReader("a,1\nb,2\nc,3\n"))
	rows, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	total := 0
	keys := []string{}
	for _, row := range rows {
		v, err := strconv.Atoi(row[*col])
		if err != nil {
			log.Fatalf("row %v: %v", row, err)
		}
		total += v
		keys = append(keys, row[0])
	}
	sort.Strings(keys)
	log.Printf("%s: %s = %d", filepath.Base(*name), strings.Join(keys, ","), total)
	fmt.Println("done")
}
