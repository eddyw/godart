package load

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// coverPackage writes a counter into each statement of the package under test.
//
// `go tool cover` of the Go tool chain does the rewrite. The tool writes 1
// counter variable for each file, and the name of that variable is a name of
// this loader. The variable is exported, thus the entry point of the test
// binary reads it and reports the coverage.
//
// The rewrite touches the files of the package alone. A test file keeps its
// source, exactly as `go test -cover` does.
func coverPackage(bin *Binary, files map[string][]byte, environment []string, variant, plain *packages.Package) error {
	names := coverFiles(variant, plain)
	if len(names) == 0 {
		return nil
	}
	// The overlay of the compiler replaces the source of each package of the
	// standard library surface. The rewrite reads the file of the disk, thus
	// such a package takes no counter and the build reports the cause.
	for _, name := range names {
		if _, held := files[name]; held {
			return fmt.Errorf("the flag -cover cannot read %s, because the overlay of gart"+
				" replaces the source of the package %s", name, bin.Under)
		}
	}
	dir, err := os.MkdirTemp("", "gartcover")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	bin.Cover = map[string][]byte{}
	for i, name := range names {
		variable := fmt.Sprintf("GartCover%d", i)
		out := filepath.Join(dir, fmt.Sprintf("%d.go", i))
		cmd := exec.Command("go", "tool", "cover", "-mode=set",
			"-var="+variable, "-o", out, name)
		cmd.Env = environment
		cmd.Dir = bin.Dir
		if text, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("go tool cover %s: %v\n%s", name, err, text)
		}
		content, err := os.ReadFile(out)
		if err != nil {
			return err
		}
		bin.Cover[name] = content
		bin.CoverVars = append(bin.CoverVars, coverVar{Name: variable, File: name})
	}
	return nil
}

// coverFiles gives the file of the package under test that the rewrite reads.
// A test file and a file that the overlay already replaced each stay out.
func coverFiles(variant, plain *packages.Package) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range []*packages.Package{variant, plain} {
		if p == nil {
			continue
		}
		for _, name := range p.GoFiles {
			if seen[name] || strings.HasSuffix(name, "_test.go") {
				continue
			}
			seen[name] = true
			out = append(out, name)
		}
		if len(out) > 0 {
			break
		}
	}
	sort.Strings(out)
	return out
}

// coverSource writes the Go source that reports the coverage of 1 test binary.
//
// The counter variable of each file holds 3 arrays: the count of each block,
// the position of each block, and the count of the statements of each block.
// The entry point maps them to the value that testing.SetCover takes.
func coverSource(bin *Binary, prefix string, counters bool) string {
	var b strings.Builder
	if !counters {
		// The package under test holds no statement. Go still reports the mode
		// of the run, thus the line of the report names no number.
		b.WriteString("\n// gartSetCover names the mode of a run with no counter.\n")
		b.WriteString("func gartSetCover() {\n")
		fmt.Fprintf(&b, "\ttesting.SetCover(testing.Cover{Mode: %q})\n", bin.CoverMode)
		b.WriteString("}\n")
		return b.String()
	}
	b.WriteString("\n// gartSetCover gives the harness the counters of the run.\n")
	b.WriteString("func gartSetCover() {\n")
	b.WriteString("\tcounters := map[string][]uint32{}\n")
	b.WriteString("\tblocks := map[string][]testing.CoverBlock{}\n")
	for _, v := range bin.CoverVars {
		name := prefix + "." + v.Name
		fmt.Fprintf(&b, "\tcounters[%q] = %s.Count[:]\n", v.File, name)
		fmt.Fprintf(&b, "\tblocks[%q] = gartCoverBlocks(%s.Pos[:], %s.NumStmt[:])\n",
			v.File, name, name)
	}
	b.WriteString("\ttesting.SetCover(testing.Cover{\n")
	fmt.Fprintf(&b, "\t\tMode:     %q,\n", bin.CoverMode)
	b.WriteString("\t\tCounters: counters,\n")
	b.WriteString("\t\tBlocks:   blocks,\n")
	b.WriteString("\t})\n")
	b.WriteString("}\n\n")
	b.WriteString("// gartCoverBlocks reads the position table of 1 file.\n")
	b.WriteString("func gartCoverBlocks(pos []uint32, stmts []uint16) []testing.CoverBlock {\n")
	b.WriteString("\tout := make([]testing.CoverBlock, len(stmts))\n")
	b.WriteString("\tfor i := range out {\n")
	b.WriteString("\t\tout[i] = testing.CoverBlock{\n")
	b.WriteString("\t\t\tLine0: pos[3*i],\n")
	b.WriteString("\t\t\tCol0:  uint16(pos[3*i+2] >> 16),\n")
	b.WriteString("\t\t\tLine1: pos[3*i+1],\n")
	b.WriteString("\t\t\tCol1:  uint16(pos[3*i+2] & 0xFFFF),\n")
	b.WriteString("\t\t\tStmts: stmts[i],\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn out\n")
	b.WriteString("}\n")
	return b.String()
}
