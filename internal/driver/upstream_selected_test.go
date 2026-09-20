package driver_test

import (
	"archive/zip"
	"encoding/json"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/parity"
)

func TestSelectedUpstream(t *testing.T) {
	var inventory parity.Inventory
	inventoryName, err := parity.InventoryName(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join("..", "parity", "testdata", inventoryName))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &inventory); err != nil {
		t.Fatal(err)
	}
	selections := map[string][]upstreamFile{
		"fmt": {
			{Name: "scan_test.go", Declarations: []string{"RecursiveInt", "RecursiveInt.Scan", "scanInts", "makeInts", "TestScanInts", "intCount", "testScanInts", "f"}},
			{Name: "fmt_test.go", Declarations: []string{"TestComplexFormatting", "NaN", "posInf", "negInf"}},
		},
		"runtime": {
			{Name: "proc_test.go", Declarations: []string{"preempt", "TestPreemption", "TestTimerFairness"}},
			{Name: "defer_test.go", Declarations: []string{
				"TestUnconditionalPanic", "glob", "TestOpenAndNonOpenDefers", "testOpen", "TestNonOpenAndOpenDefers",
				"list", "TestConditionalDefers", "testConditionalDefers", "TestDisappearingDefer", "TestAbortedPanic", "TestRecoverMatching",
				"nonSSAable", "bigStruct", "containsBigStruct", "mknonSSAable", "globint1", "globint2", "globint3", "sideeffect", "sideeffect2", "TestNonSSAableArgs",
				"TestDeferWithRepeatedRepanics", "interpreter", "recurseFnPanicRec", "saveInt", "recurseFn",
			}},
		},
		"syscall":          {{Name: "syscall_test.go", Declarations: []string{"testSetGetenv", "TestEnv"}}, {Name: "env_unix_test.go", Declarations: []string{"env", "genDummyEnv", "setDummyEnv", "setupEnvCleanup", "TestClearenv"}}},
		"internal/godebug": {{Name: "godebug_test.go", Declarations: []string{"TestGet", "TestImmutable"}}},
		"cmp":              {{Name: "cmp_test.go", Declarations: []string{"negzero", "TestSort", "TestOr", "ExampleOr", "ExampleOr_sort", "ExampleLess", "ExampleCompare"}}},
		"math":             {{Name: "const_test.go"}, {Name: "example_test.go"}},
		"encoding/asn1": {
			{Name: "marshal_test.go"},
			{Name: "asn1_test.go", Declarations: []string{"TestUnmarshalNestingLimitSlice", "TestUnmarshalNestingLimitStruct"}},
		},
		"crypto/tls": {
			{Name: "tls_test.go", Declarations: []string{"TestX509KeyPair", "TestX509KeyPairPopulateCertificate", "testingKey"}},
			{Name: "certificates_test.go", Declarations: []string{"testMLDSA44CertPEM", "testMLDSA44KeyPEM", "testMLDSA65CertPEM", "testMLDSA65KeyPEM", "testMLDSA87CertPEM", "testMLDSA87KeyPEM"}},
		},
		"crypto/x509": {
			{Name: "verify_test.go", Declarations: []string{
				"TestGoVerify",
				"certificateFromPEM",
				"chainToDebugString",
				"criticalExtIntermediate",
				"criticalExtIntermediateWithExt",
				"criticalExtLeaf",
				"criticalExtLeafWithExt",
				"criticalExtRoot",
				"digicertRoot",
				"excludedNamesIntermediate",
				"excludedNamesLeaf",
				"excludedNamesRoot",
				"expectAuthorityUnknown",
				"expectExpired",
				"expectHashError",
				"expectHostnameError",
				"expectNameConstraintsError",
				"expectNotAuthorizedError",
				"expectUnhandledCriticalExtension",
				"expectUsageError",
				"generatePEMCertWithRepeatSAN",
				"globalSignRoot",
				"googleLeaf",
				"googleLeafWithInvalidHash",
				"gtsIntermediate",
				"gtsRoot",
				"ignoreCNWithSANLeaf",
				"ignoreCNWithSANRoot",
				"invalidCNRoot",
				"leafMatchingAKIDMatchingIssuer",
				"leafWithAKID",
				"nameConstraintsIntermediate1",
				"nameConstraintsIntermediate2",
				"nameConstraintsLeaf",
				"nameToKey",
				"rootMatchingSKIDMismatchingSubject",
				"rootMismatchingSKIDMatchingSubject",
				"rootWithoutSKID",
				"selfSigned",
				"smimeIntermediate",
				"smimeLeaf",
				"smimeRoot",
				"testVerify",
				"trustAsiaLeaf",
				"trustAsiaSHA384Intermediate",
				"validCNWithoutSAN",
				"verifyTest",
				"verifyTests",
				"x509v1TestIntermediate",
				"x509v1TestLeaf",
				"x509v1TestRoot",
			}},
			{Name: "pkcs8_test.go"}, {Name: "sec1_test.go"}, {Name: "pem_decrypt_test.go"},
			{Name: "x509_test.go", Declarations: []string{
				"TestPKCS1MismatchPublicKeyFormat", "TestMarshalInvalidPublicKey", "TestPKIXMismatchPublicKeyFormat",
				"testParsePKIXPublicKey", "TestParsePKIXPublicKey", "pemPublicKey", "pemEd25519Key", "pemX25519Key",
				"TestCertificateParse", "certBytes", "TestCertificateEqualOnNil",
				"TestMismatchedSignatureAlgorithm", "rsaPSSSelfSignedPEM",
				"TestMLDSA", "testMLDSA", "pemDecode",
				"rfc9881ExamplePrivateKeyMLDSA44", "rfc9881ExamplePrivateKeyMLDSA44Expanded", "rfc9881ExamplePrivateKeyMLDSA44Both",
				"rfc9881ExamplePrivateKeyMLDSA65", "rfc9881ExamplePrivateKeyMLDSA87",
				"rfc9881ExamplePublicKeyMLDSA44", "rfc9881ExamplePublicKeyMLDSA65", "rfc9881ExamplePublicKeyMLDSA87",
				"rfc9881ExampleCertificateMLDSA44", "rfc9881ExampleCertificateMLDSA65", "rfc9881ExampleCertificateMLDSA87",
			}},
		},
		"runtime/debug": {{Name: "mod_test.go"}},
		"os/exec":       {{Name: "env_test.go"}},
		"sync": {
			{Name: "once_test.go"}, {Name: "cond_test.go"},
			{Name: "mutex_test.go", Declarations: []string{"misuseTests", "init", "TestMutexMisuse", "TestMutexFairness"}},
			{Name: "rwmutex_test.go"},
			{Name: "oncefunc_test.go", Declarations: []string{
				"testOncePanicX", "testOncePanicWith", "TestOnceFuncPanic", "TestOnceValuePanic",
				"TestOnceValuesPanic", "TestOnceFuncPanicNil", "TestOnceFuncGoexit",
				"TestOnceFuncPanicTraceback", "onceFuncPanic",
			}},
			{Name: "waitgroup_test.go", Declarations: []string{"testWaitGroup", "TestWaitGroup", "TestWaitGroupMisuse", "TestWaitGroupRace", "TestWaitGroupAlign", "TestWaitGroupGo", "TestIssue76126"}},
		},
		"sync/atomic": {{Name: "value_test.go"}, {Name: "atomic_test.go", Declarations: atomicDeclarations()}},
		"reflect": {
			{Name: "visiblefields_test.go"},
			{Name: "iter_test.go"},
			{Name: "type_test.go", Declarations: []string{"TestTypeFor", "TestType_CanSeq", "TestType_CanSeq2"}},
			{Name: "example_test.go", Declarations: []string{"ExampleStructTag_Lookup"}},
			{Name: "set_test.go", Declarations: []string{"TestImplicitSetConversion", "TestImplicitSendConversion", "TestImplicitCallConversion", "TestImplicitAppendConversion"}},
			{Name: "all_test.go", Declarations: []string{
				"shouldPanic", "TestTypeOverflow", "TestInitFuncTypes",
				"TestBool", "TestCanIntUintFloatComplex", "TestCanSetField",
				"TestPtrSetNil", "TestMapSetNil", "appendTests", "TestAppend",
				"TestCopy", "TestCopyString", "TestCopyArray", "TestBigUnnamedStruct",
				"iterateToString", "TestMapIterNonEmptyMap", "TestMapIterNilMap",
				"TestMapIterSafety", "TestMapIterNext", "TestMapIterDelete0", "TestMapIterDelete1",
				"TestValue_Cap", "TestValue_Len",
				"assert", "TestInterfaceGet", "TestInterfaceValue", "TestFunctionValue",
				"two", "dummy", "TestFunc", "TestCallConvert", "TestMakeFunc",
				"TestMakeFuncInterface", "TestMakeFuncVariadic", "TestSetIter",
				"NonComparableStruct", "TestValue_Comparable", "TestValue_EqualNonComparable",
				"ValueEqualTest", "equalI", "nilInterface", "valueEqualTests", "TestValue_Equal",
				"TestSetPanic", "TestSetBytes",
			}},
		},
		"time": {{Name: "format_test.go", Declarations: []string{"TimeFormatTest", "rfc3339Formats", "TestRFC3339Conversion", "FormatTest", "formatTests", "TestFormat", "goStringTests", "TestGoString", "TestFormatSingleDigits", "TestFormatShortYear", "TestNoonIs12PM", "TestMidnightIs12AM", "Test12PMIsNoon", "Test12AMIsMidnight"}}, {Name: "example_test.go"}, {Name: "time_test.go", Declarations: []string{
			"durationTests", "TestDurationString", "parseDurationTests", "TestParseDuration", "parseDurationErrorTests", "TestParseDurationErrors", "TestParseDurationRoundTrip",
			"nsDurationTests", "TestDurationNanoseconds", "usDurationTests", "TestDurationMicroseconds", "msDurationTests", "TestDurationMilliseconds",
			"secDurationTests", "TestDurationSeconds", "minDurationTests", "TestDurationMinutes", "hourDurationTests", "TestDurationHours",
			"durationTruncateTests", "TestDurationTruncate", "durationRoundTests", "TestDurationRound", "durationAbsTests", "TestDurationAbs", "minDuration", "maxDuration",
		}}},
		"os": {
			{Name: "env_test.go"}, {Name: "env_unix_test.go"}, {Name: "error_test.go"},
			{Name: "read_test.go", Declarations: []string{"checkNamedSize", "TestReadFile", "TestWriteFile", "TestReadOnlyWriteFile"}},
			{Name: "tempfile_test.go", Declarations: []string{"TestCreateTemp", "TestCreateTempPattern", "TestMkdirTemp", "TestMkdirTempBadDir"}},
			{Name: "removeall_test.go", Declarations: []string{"testRequiresPermissions", "TestRemoveAll", "TestRemoveAllLarge", "TestRemoveAllLongPath", "TestRemoveAllDot", "TestRemoveAllDotDot", "TestRemoveReadOnlyDir", "TestRemoveAllButReadOnlyAndPathError", "TestRemoveUnreadableDir", "TestRemoveAllWithMoreErrorThanReqSize", "TestRemoveAllTrailingSlash", "TestRemoveAllSymlinkRemovalFailure"}},
			{Name: "path_test.go", Declarations: []string{"TestMkdirAll", "TestMkdirAllAbsPath"}},
			{Name: "root_test.go", Declarations: []string{"testMaybeRooted", "makefs", "hasLink", "rootTest", "rootTest.run", "rootTestCases", "errEndsTest", "TestRootMkdirAll", "TestRootMkdir", "TestRootOpen_File", "TestRootOpen_Directory", "TestRootCreate", "TestRootOpenRoot", "TestRootStat", "TestRootLstat", "TestRootReadlink", "TestRootRemoveFile", "TestRootRemoveDirectory", "TestRootNonPermissionMode", "TestRootUseAfterClose", "TestRootSymlinkToRoot", "TestOpenInRoot", "TestRootRemoveDot", "TestRootWriteReadFile", "TestRootName", "TestRootConcurrentClose"}},
		},
		"net":      {{Name: "ip_test.go"}, {Name: "mac_test.go"}},
		"net/http": {{Name: "header_test.go", Declarations: []string{"headerWriteTests", "TestHeaderWrite", "parseTimeTests", "TestParseTime", "TestNilHeaderClone"}}, {Name: "sniff_test.go", Declarations: []string{"sniffTests", "TestDetectContentType"}}},
	}
	for pkg, files := range selections {
		t.Run(pkg, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(runtime.GOROOT(), "src", filepath.FromSlash(pkg))
			fixture := t.TempDir()
			writeFile(t, filepath.Join(fixture, "go.mod"), "module example.com/upstream\n\ngo 1.27\n")
			if pkg == "sync" || pkg == "crypto/x509" || pkg == "os" {
				module := "example.com/upstream"
				if pkg == "sync" {
					module = "sync_test"
				}
				dependencies := "require example.com/testenv v0.0.0\nreplace example.com/testenv => ./testenv\n"
				if pkg == "crypto/x509" {
					dependencies += "\nrequire example.com/cryptotest v0.0.0\nreplace example.com/cryptotest => ./cryptotest\n"
				}
				writeFile(t, filepath.Join(fixture, "go.mod"), "module "+module+"\n\ngo 1.27\n\n"+dependencies)
			}
			var certificates *certificateInputs
			if pkg == "crypto/x509" {
				certificates = prepareCertificateInputs(t, filepath.Join(dir, "verify_test.go"))
				if err := os.MkdirAll(filepath.Join(fixture, "cryptotest"), 0755); err != nil {
					t.Fatal(err)
				}
				source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/crypto/internal/cryptotest/fips140.go"), []string{"MustMinimumFIPS140ModuleVersion", "parseFIPS140MinorVersion"})
				source = strings.ReplaceAll(source, `"crypto/internal/fips140"`, `"crypto/fips140"`)
				writeFile(t, filepath.Join(fixture, "cryptotest/go.mod"), "module example.com/cryptotest\n\ngo 1.27\n")
				writeFile(t, filepath.Join(fixture, "cryptotest/fips140.go"), source)
			}
			selectedFiles := map[string]upstreamFile{}
			for _, file := range files {
				selectedFiles[file.Name] = file
				source := selectedUpstreamSource(t, filepath.Join(dir, file.Name), file.Declarations)
				if certificates != nil && file.Name == "verify_test.go" {
					source = applyCertificateInputs(t, source, certificates.Inputs)
				}
				if pkg == "sync" {
					// The original traceback assertion requires the sync_test import path.
					source = strings.Replace(source, "package sync_test", "package sync", 1)
					source = strings.ReplaceAll(source, `"internal/testenv"`, `"example.com/testenv"`)
				}
				if pkg == "net" {
					source = strings.Replace(source, "package net", "package net_test\nimport . \"net\"", 1)
				}
				if pkg == "net/http" && file.Name == "header_test.go" {
					source = strings.Replace(source, "package http", "package http_test\nimport . \"net/http\"", 1)
				}
				if pkg == "crypto/tls" {
					source = strings.Replace(source, "package tls", "package tls_test", 1)
					if file.Name == "tls_test.go" {
						source = strings.Replace(source, "package tls_test", "package tls_test\nimport . \"crypto/tls\"", 1)
					}
				}
				if pkg == "crypto/x509" {
					source = strings.ReplaceAll(source, `"crypto/internal/cryptotest"`, `"example.com/cryptotest"`)
					source = strings.ReplaceAll(source, `"internal/testenv"`, `"example.com/testenv"`)
					source = strings.Replace(source, "package x509", "package x509_test\nimport . \"crypto/x509\"", 1)
				}
				if pkg == "os" {
					source = strings.ReplaceAll(source, `"internal/testenv"`, `"example.com/testenv"`)
				}
				if pkg == "encoding/asn1" {
					source = strings.Replace(source, "package asn1", "package asn1_test\nimport . \"encoding/asn1\"", 1)
				}
				if pkg == "internal/godebug" {
					source = strings.ReplaceAll(source, `"internal/godebug"`, `"example.com/upstream"`)
				}
				writeFile(t, filepath.Join(fixture, file.Name), source)
			}
			if pkg == "sync" || pkg == "crypto/x509" || pkg == "os" {
				names := []string{"Executable", "MustHaveExec"}
				if pkg == "crypto/x509" {
					names = []string{"Builder", "SkipFlaky"}
				} else if pkg == "os" {
					names = []string{"MustHaveSymlink"}
				}
				source := selectedUpstreamSource(t, "../overlay/std/internal/testenv/testenv.go", names)
				if err := os.MkdirAll(filepath.Join(fixture, "testenv"), 0755); err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(fixture, "testenv/go.mod"), "module example.com/testenv\n\ngo 1.27\n")
				writeFile(t, filepath.Join(fixture, "testenv/testenv.go"), strings.TrimPrefix(source, "//go:build dart\n"))
			}

			if pkg == "net" {
				source := selectedUpstreamSource(t, "../overlay/std/net/addr.go", []string{"networkNumberAndMask"})
				source = strings.Replace(source, "package net", "package net_test\nimport . \"net\"", 1)
				source = strings.TrimPrefix(source, "//go:build dart\n")
				writeFile(t, filepath.Join(fixture, "helper_test.go"), source)
			}

			if pkg == "os/exec" {
				source := selectedUpstreamSource(t, "../overlay/std/os/exec/exec.go", []string{"dedupEnvCase"})
				writeFile(t, filepath.Join(fixture, "helper.go"), source)
			}
			if pkg == "internal/godebug" {
				data, err := os.ReadFile("../overlay/std/internal/godebug/godebug.go")
				if err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(fixture, "godebug.go"), string(data))
			}

			if pkg == "time" {
				zones, err := zip.OpenReader(filepath.Join(runtime.GOROOT(), "lib/time/zoneinfo.zip"))
				if err != nil {
					t.Fatal(err)
				}
				defer zones.Close()
				for _, zone := range zones.File {
					if zone.Name != "America/Los_Angeles" {
						continue
					}
					r, err := zone.Open()
					if err != nil {
						t.Fatal(err)
					}
					data, err := io.ReadAll(r)
					r.Close()
					if err != nil {
						t.Fatal(err)
					}
					writeFile(t, filepath.Join(fixture, "000_zone_test.go"), "package time_test\nimport \"time\"\nvar _ = func() bool { loc, err := time.LoadLocationFromTZData(\"Local\", []byte("+strconv.Quote(string(data))+")); if err != nil { panic(err) }; *time.Local = *loc; return true }()\n")
				}
			}

			var names []string
			for _, p := range inventory.Packages {
				if p.Path != pkg {
					continue
				}
				for _, test := range p.Tests {
					file, ok := selectedFiles[test.File]
					if ok && (len(file.Declarations) == 0 || slices.Contains(file.Declarations, test.Name)) && test.Eligible && test.Kind != "Benchmark" {
						names = append(names, regexp.QuoteMeta(test.Name))
						if test.Kind == "Fuzz" {
							corpus := filepath.Join("testdata", "fuzz", test.Name)
							if _, err := os.Stat(filepath.Join(dir, corpus)); err == nil {
								if err := os.CopyFS(filepath.Join(fixture, corpus), os.DirFS(filepath.Join(dir, corpus))); err != nil {
									t.Fatal(err)
								}
							} else if !os.IsNotExist(err) {
								t.Fatal(err)
							}
						}
					}
				}
			}
			if len(names) == 0 {
				t.Fatal("empty upstream file selection")
			}
			filter := "^(" + strings.Join(names, "|") + ")$"
			out := t.TempDir()
			stats, err := driver.BuildWithStats(driver.Options{Dir: fixture, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
			if err != nil {
				t.Fatal(err)
			}
			if len(stats.Binaries) != 1 {
				t.Fatalf("got %d binaries", len(stats.Binaries))
			}
			prepareDart(t, out)
			var launcher string
			if pkg == "sync" {
				launcher = prepareUpstreamSelfExec(t, out, stats.Binaries[0].Entry)
			}
			args := []string{"-test.v", "-test.short=" + strconv.FormatBool(testing.Short()), "-test.run=" + filter, "-test.skip=" + upstreamSkip(t, pkg)}
			result := programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			name := pkg + "-selected"
			if pkg == "net/http" {
				name = pkg + "-content-type"
			}
			compareUpstreamCertificates(t, name, pkg, filter, "vm", dir, result.Text, result.Exit, certificates)
			if pkg == "reflect" || pkg == "crypto/x509" || pkg == "crypto/tls" || pkg == "runtime" || pkg == "fmt" || pkg == "encoding/asn1" {
				for _, target := range []string{"aot", "js", "wasm"} {
					t.Run(target, func(t *testing.T) {
						result := runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
						compareUpstreamCertificates(t, name, pkg, filter, target, dir, result.Text, result.Exit, certificates)
					})
				}
			}
			if pkg == "os" {
				t.Run("aot", func(t *testing.T) {
					result := runUpstreamTarget(t, "aot", out, stats.Binaries[0].Entry, dir, args)
					compareUpstream(t, name, pkg, filter, "aot", dir, result.Text, result.Exit)
				})
			}
			if pkg == "sync" {
				run(t, out, "dart", "compile", "exe", stats.Binaries[0].Entry, "-o", launcher)
				result := programOutcome(t, dir, launcher, args...)
				compareUpstream(t, name, pkg, filter, "aot", dir, result.Text, result.Exit)
			}
		})
	}
}

type upstreamFile struct {
	Name         string
	Declarations []string
}

func atomicDeclarations() []string {
	names := []string{"magic32", "magic64", "testCompareAndSwapUint64"}
	for _, operation := range []string{"Swap", "Add", "And", "Or", "CompareAndSwap", "Load", "Store"} {
		for _, typ := range []string{"Int32", "Uint32", "Int64", "Uint64"} {
			names = append(names, "Test"+operation+typ, "Test"+operation+typ+"Method")
		}
	}
	return names
}

// Keep original declaration bytes. Only imports and the package wrapper adapt.
func selectedUpstreamSource(t *testing.T, path string, names []string) string {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, path, source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var declarations []ast.Decl
	used := map[string]bool{}
	found := map[string]bool{}
	for _, decl := range f.Decls {
		keep := len(names) == 0
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if strings.HasPrefix(d.Name.Name, "Benchmark") {
				continue
			}
			candidates := []string{d.Name.Name}
			if d.Recv != nil {
				receiver := d.Recv.List[0].Type
				if pointer, ok := receiver.(*ast.StarExpr); ok {
					receiver = pointer.X
				}
				if name, ok := receiver.(*ast.Ident); ok {
					candidates = append(candidates, name.Name+"."+d.Name.Name)
				}
			}
			for _, name := range candidates {
				if slices.Contains(names, name) {
					keep = true
					found[name] = true
				}
			}
		case *ast.GenDecl:
			if d.Tok == token.IMPORT {
				continue
			}
			for _, spec := range d.Specs {
				switch v := spec.(type) {
				case *ast.ValueSpec:
					for _, n := range v.Names {
						if slices.Contains(names, n.Name) {
							keep = true
							found[n.Name] = true
						}
					}
				case *ast.TypeSpec:
					if slices.Contains(names, v.Name.Name) {
						keep = true
						found[v.Name.Name] = true
					}
				}
			}
		}
		if !keep {
			continue
		}
		declarations = append(declarations, decl)
		ast.Inspect(decl, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if id, ok := sel.X.(*ast.Ident); ok {
					used[id.Name] = true
				}
			}
			return true
		})
	}
	for _, name := range names {
		if !found[name] {
			t.Fatalf("missing upstream declaration %s in %s", name, path)
		}
	}
	var out strings.Builder
	out.Write(source[:fs.Position(f.Name.End()).Offset])
	out.WriteString("\nimport (\n")
	for _, imp := range f.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		name := filepath.Base(path)
		if imp.Name != nil {
			name = imp.Name.Name
		}
		if name == "." || name == "_" || used[name] {
			out.Write(source[fs.Position(imp.Pos()).Offset:fs.Position(imp.End()).Offset])
			out.WriteByte('\n')
		}
	}
	out.WriteString(")\n")
	for _, decl := range declarations {
		start := decl.Pos()
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Doc != nil {
				start = d.Doc.Pos()
			}
		case *ast.GenDecl:
			if d.Doc != nil {
				start = d.Doc.Pos()
			}
		}
		out.Write(source[fs.Position(start).Offset:fs.Position(decl.End()).Offset])
		out.WriteString("\n\n")
	}
	return out.String()
}
