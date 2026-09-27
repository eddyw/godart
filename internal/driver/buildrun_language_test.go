package driver_test

import (
	"encoding/json"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/parity"
)

func TestBuildRunChildOfGoTest(t *testing.T) {
	inv, err := parity.DiscoverLanguage(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	var outcomes []*languageOutcome
	for _, c := range inv.Cases {
		if c.Adapter == "child-buildrun" {
			outcomes = append(outcomes, &languageOutcome{Fixture: c, Contract: "buildrun", Status: "unobserved"})
		}
	}
	retainLanguageEvidence(t, "vm", inv, outcomes)
	for _, result := range outcomes {
		t.Run(result.Fixture.Name, func(t *testing.T) {
			t.Cleanup(func() {
				switch {
				case t.Failed():
					result.Status = "failed"
				case t.Skipped():
					result.Status = "excluded"
				default:
					result.Status = "passed"
				}
			})
			c := result.Fixture
			if !c.NativeEligible || !c.GartEligible {
				t.Skipf("upstream build constraints: native=%t Gart=%t", c.NativeEligible, c.GartEligible)
			}
			source, err := os.ReadFile(filepath.Join(build.Default.GOROOT, "test", c.File))
			if err != nil {
				t.Fatal(err)
			}
			file, err := parser.ParseFile(token.NewFileSet(), c.File, source, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, declaration := range file.Decls {
				decl, ok := declaration.(*ast.GenDecl)
				if !ok || decl.Tok != token.CONST {
					continue
				}
				for _, specification := range decl.Specs {
					value := specification.(*ast.ValueSpec)
					if len(value.Names) == 1 && value.Names[0].Name == "prog" {
						if len(value.Values) != 1 {
							t.Fatal("upstream child initializer differs")
						}
						literal, ok := value.Values[0].(*ast.BasicLit)
						if !ok || literal.Kind != token.STRING {
							t.Fatal("upstream child is not a string literal")
						}
						result.GeneratedSource, err = strconv.Unquote(literal.Value)
						if err != nil {
							t.Fatal(err)
						}
					}
				}
			}
			if result.GeneratedSource == "" {
				t.Fatal("upstream child source is absent")
			}
			dir := t.TempDir()
			result.AdaptedSources = map[string]string{"fixture.go": string(source)}
			writeFile(t, filepath.Join(dir, "fixture.go"), string(source))
			result.ModuleSource = "module example.com/buildrunfixture\n\ngo " + strings.TrimPrefix(inv.GoVersion, "go") + "\n"
			result.ModuleSHA256 = languageDigest([]byte(result.ModuleSource))
			writeFile(t, filepath.Join(dir, "go.mod"), result.ModuleSource)
			license, err := os.ReadFile(filepath.Join(build.Default.GOROOT, "LICENSE"))
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
			compiler := filepath.Join(t.TempDir(), "gart.exe")
			run(t, "", "go", "build", "-o", compiler, "../../cmd/gart")
			native := filepath.Join(t.TempDir(), "native")
			result.NativeCompile = make([]languageExecution, 1)
			runLanguage(t, &result.NativeCompile[0], dir, runLimit, "go", "build", "-gcflags=all="+inv.GCFlags, "-o", native, "fixture.go")
			out := t.TempDir()
			if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
				result.BuildError = err.Error()
				t.Fatal(err)
			}
			prepareDart(t, out)
			goPath, err := exec.LookPath("go")
			if err != nil {
				t.Fatal(err)
			}
			proxy := t.TempDir()
			result.WrapperSource = buildRunGoProxy
			result.WrapperSHA256 = languageDigest([]byte(buildRunGoProxy))
			writeFile(t, filepath.Join(proxy, "go"), buildRunGoProxy)
			if err := os.Chmod(filepath.Join(proxy, "go"), 0755); err != nil {
				t.Fatal(err)
			}
			settings := map[string]string{"PATH": proxy + string(os.PathListSeparator) + os.Getenv("PATH")}
			for _, target := range []string{"native", "gart"} {
				config := map[string]any{"go": goPath, "gart": compiler, "runtime": runtimeDir(t), "output": t.TempDir(), "target": target, "limit": runLimit.Seconds()}
				data, err := json.Marshal(config)
				if err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(proxy, "config.json"), string(data))
				if target == "native" {
					result.Native.Environment = settings
					runLanguage(t, &result.Native, dir, runLimit, native)
				} else {
					result.Gart.Environment = settings
					runLanguage(t, &result.Gart, dir, runLimit, "dart", filepath.Join(out, "bin/main.dart"))
				}
				data, err = os.ReadFile(filepath.Join(proxy, "record.json"))
				if err != nil {
					t.Fatal(err)
				}
				var record struct {
					Source    string            `json:"source"`
					Output    string            `json:"output"`
					Execution languageExecution `json:"execution"`
				}
				if err := json.Unmarshal(data, &record); err != nil {
					t.Fatal(err)
				}
				if record.Source != result.GeneratedSource || !record.Execution.Ran || record.Execution.TimedOut || record.Execution.Exit == 0 {
					t.Fatalf("child execution differs: %+v", record.Execution)
				}
				if record.Execution.OutputSHA256 != languageDigest([]byte(record.Output)) {
					t.Fatal("child output digest differs")
				}
				if target == "native" {
					result.NativeChild, result.NativeDiagnostics = &record.Execution, record.Output
				} else {
					result.GartChild, result.GartDiagnostics = &record.Execution, record.Output
				}
				if err := os.Remove(filepath.Join(proxy, "record.json")); err != nil {
					t.Fatal(err)
				}
			}
			result.ExpectedSHA256 = languageDigest(nil)
			if result.Native.OutputSHA256 != result.ExpectedSHA256 || result.Gart.OutputSHA256 != result.ExpectedSHA256 {
				t.Fatal("upstream parent output differs")
			}
		})
	}
}

const buildRunGoProxy = `#!/usr/bin/env python3
import hashlib,json,os,subprocess,sys,time
from pathlib import Path
base=Path(__file__).resolve().parent
c=json.loads((base/'config.json').read_text())
args=sys.argv[1:]
if not args or args[0]!='run':os.execv(c['go'],[c['go']]+args)
if len(args)!=2 or not args[1].endswith('.go'):raise SystemExit('unexpected upstream child command')
source=Path(args[1]).read_text()
command=[c['go']]+args if c['target']=='native' else [c['gart'],'run','-frontend=source','-format=false','-o',c['output'],'-runtime',c['runtime'],args[1]]
start=time.monotonic()
try:
 result=subprocess.run(command,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,timeout=c['limit'])
 code,output,timed_out=result.returncode,result.stdout,False
except subprocess.TimeoutExpired as error:
 code,output,timed_out=124,error.stdout or b'',True
record=dict(source=source,output=output.decode(),execution=dict(command=command,ran=True,exit=code,timed_out=timed_out,seconds=time.monotonic()-start,output_sha256=hashlib.sha256(output).hexdigest()))
with (base/'record.json').open('x') as file:json.dump(record,file)
sys.stdout.buffer.write(output)
sys.stdout.buffer.flush()
sys.exit(code)
`
