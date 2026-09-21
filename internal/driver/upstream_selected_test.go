package driver_test

import (
	"archive/zip"
	"encoding/json"
	"fmt"
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
	"github.com/eddyw/godart/internal/load"
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
		"os/signal": {{Name: "signal_test.go", Declarations: []string{"TestNotifyContextStop", "TestNotifyContextCancelParent", "TestNotifyContextPrematureCancelParent", "TestNotifyContextSimultaneousStop", "TestNotifyContextStringer", "TestIgnored"}}},
		"fmt": {
			{Name: "scan_test.go", Declarations: []string{"RecursiveInt", "RecursiveInt.Scan", "scanInts", "makeInts", "TestScanInts", "intCount", "testScanInts", "f"}},
			{Name: "fmt_test.go", Declarations: []string{"TestComplexFormatting", "NaN", "posInf", "negInf"}},
		},
		"context": {{Name: "x_test.go", Declarations: []string{"TestContextErrDoneRace"}}},
		"testing": {{Name: "benchmark_test.go", Declarations: []string{"TestResultString"}}},
		"go/parser": {
			{Name: "parser.go", Declarations: []string{"maxNestLev"}},
			{Name: "resolver.go", Declarations: []string{"maxScopeDepth"}},
			{Name: "parser_test.go", Declarations: []string{"parseDepthTests", "split", "TestParseDepthLimit", "TestScopeDepthLimit"}},
		},
		"html/template": {{Name: "exec_test.go", Declarations: []string{"TestMaxExecDepth"}}},
		"runtime": {
			{Name: "symtab_test.go", Declarations: []string{"TestNilName"}},
			{Name: "crash_test.go", Declarations: []string{"TestGoexitDefer"}},
			{Name: "proc_test.go", Declarations: []string{"preempt", "TestPreemption", "TestTimerFairness"}},
			{Name: "defer_test.go", Declarations: []string{
				"TestUnconditionalPanic", "glob", "TestOpenAndNonOpenDefers", "testOpen", "TestNonOpenAndOpenDefers",
				"list", "TestConditionalDefers", "testConditionalDefers", "TestDisappearingDefer", "TestAbortedPanic", "TestRecoverMatching",
				"nonSSAable", "bigStruct", "containsBigStruct", "mknonSSAable", "globint1", "globint2", "globint3", "sideeffect", "sideeffect2", "TestNonSSAableArgs",
				"TestDeferWithRepeatedRepanics", "interpreter", "recurseFnPanicRec", "saveInt", "recurseFn",
			}},
		},
		"syscall":          {{Name: "syscall_test.go", Declarations: []string{"testSetGetenv", "TestEnv"}}, {Name: "env_unix_test.go", Declarations: []string{"env", "genDummyEnv", "setDummyEnv", "setupEnvCleanup", "TestClearenv"}}},
		"internal/godebug": {{Name: "godebug_test.go", Declarations: []string{"TestGet", "TestMetrics", "TestImmutable", "TestCmdBisect", "TestBisectTestCase"}}},
		"cmp":              {{Name: "cmp_test.go", Declarations: []string{"negzero", "TestSort", "TestOr", "ExampleOr", "ExampleOr_sort", "ExampleLess", "ExampleCompare"}}},
		"math":             {{Name: "const_test.go"}, {Name: "example_test.go"}},
		"encoding/asn1": {
			{Name: "marshal_test.go"},
			{Name: "asn1_test.go", Declarations: []string{"TestUnmarshalNestingLimitSlice", "TestUnmarshalNestingLimitStruct"}},
		},
		"crypto/md5": {{Name: "md5_test.go", Declarations: []string{"md5Test", "golden", "TestGolden", "TestGoldenMarshal", "TestLarge", "TestExtraLarge", "unmarshalTest", "largeUnmarshalTests", "safeSum", "TestLargeHashes"}}},
		"crypto/subtle": {
			{Name: "constant_time_test.go"},
			{Name: "xor_test.go", Declarations: []string{"TestXORBytes", "TestXorBytesPanic", "mustPanic"}},
		},
		"crypto/tls": {
			{Name: "tls_test.go", Declarations: []string{"TestX509KeyPair", "TestX509KeyPairPopulateCertificate", "testingKey", "TestCloneFuncFields", "TestCloneNilConfig", "TestVersionName"}},
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
		"runtime/debug":   {{Name: "mod_test.go"}},
		"runtime/metrics": {{Name: "description_test.go", Declarations: []string{"TestReadEmptySlice"}}},
		"os/exec":         {{Name: "env_test.go"}},
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
			{Name: "set_test.go", Declarations: []string{"TestImplicitSetConversion", "TestImplicitSendConversion", "TestImplicitCallConversion", "TestImplicitAppendConversion", "implementsTests", "notAnExpr", "notAnExpr.Pos", "notAnExpr.End", "notAnExpr.exprNode", "notASTExpr", "TestImplements", "assignableTests", "IntPtr", "IntPtr1", "Ch", "TestAssignableTo"}},
			{Name: "all_test.go", Declarations: []string{
				"TestTagGet", "tagGetTests", "TestFuncArg", "TestStructArg", "TestZeroSet", "TestInvalid", "TestLarge", "Impl", "Impl.F", "TestValueString", "TestSlice", "TestSlice3", "TestSetLenCap", "TestVariadic", "emptyStruct", "TestCallReturnsEmpty", "fieldIndexRecover", "TestFieldByIndexNil", "TestTypeFieldOutOfRangePanic", "TestTypeFieldReadOnly", "shouldPanic", "TestBytes", "TestValuePanic", "TestTypeOverflow", "TestInitFuncTypes",
				"TestBool", "TestCanIntUintFloatComplex", "TestCanSetField", "Nil", "NotNil", "TestIsNil", "TestPtrTo",
				"T", "pair", "integer", "valueTests", "TestClear", "TestGrow", "TestSwapper", "TestPtrSetNil", "TestMapSetNil", "appendTests", "TestAppend",
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
				"WC", "WC.Write", "WC.Close", "TestMakeFuncValidReturnAssignments", "TestMakeFuncInvalidReturnAssignments",
				"Point", "Point.AnotherMethod", "Point.Dist", "Point.GCMethod", "Point.NoArgs", "Point.TotalDist", "Point.Int64Method", "Point.Int32Method",
				"TestMethod", "TestVariadicMethodValue", "TestInterfaceSet",
				"DirectIfaceT", "DirectIfaceT.M", "TestDirectIfaceMethod",
				"Tinter", "Tsmallv", "Tsmallv.M", "Tsmallp", "Tsmallp.M", "Twordv", "Twordv.M", "Twordp", "Twordp.M",
				"Tbigv", "Tbigv.M", "Tbigp", "Tbigp.M", "Tm1", "Tm2", "Tm3", "Tm4", "Tm4.M", "TestMethod5",
				"unexp", "unexp.f", "unexp.g", "unexpI", "TestUnexportedMethods", "TestUnexported", "Private", "Private.m", "private", "private.P", "Public", "Public.M", "isNonNil", "isValid", "timp", "timp.W", "timp.Y", "timp.w", "timp.y", "TestCallPanic", "UnExportedFirst", "UnExportedFirst.ΦExported", "UnExportedFirst.unexported", "TestMethodByNameUnExportedFirst", "TestMethodPkgPath", "TestVariadicType",
				"FuncDDD", "FuncDDD.M", "TestNumMethodOnDDD", "TestConvertSlice2Array",
			}},
		},
		"time": {{Name: "format_test.go", Declarations: []string{"TimeFormatTest", "rfc3339Formats", "TestRFC3339Conversion", "FormatTest", "formatTests", "TestFormat", "goStringTests", "TestGoString", "TestFormatSingleDigits", "TestFormatShortYear", "TestNoonIs12PM", "TestMidnightIs12AM", "Test12PMIsNoon", "Test12AMIsMidnight"}}, {Name: "example_test.go"}, {Name: "time_test.go", Declarations: []string{
			"TestMarshalBinaryZeroTime", "durationTests", "TestDurationString", "parseDurationTests", "TestParseDuration", "parseDurationErrorTests", "TestParseDurationErrors", "TestParseDurationRoundTrip",
			"nsDurationTests", "TestDurationNanoseconds", "usDurationTests", "TestDurationMicroseconds", "msDurationTests", "TestDurationMilliseconds",
			"secDurationTests", "TestDurationSeconds", "minDurationTests", "TestDurationMinutes", "hourDurationTests", "TestDurationHours",
			"durationTruncateTests", "TestDurationTruncate", "durationRoundTests", "TestDurationRound", "durationAbsTests", "TestDurationAbs", "minDuration", "maxDuration",
		}}},
		"os": {
			{Name: "os_test.go", Declarations: []string{"checkMode", "TestChmod", "TestSameFile", "testDevNullFileInfo", "testDevNullFile", "TestDevNullFile", "TestStatDirModeExec", "TestFileChdir", "TestChdirAndGetwd", "TestProgWideChdir", "equal", "TestGetppid", "TestAppend", "writeFile", "TestSeek", "newFile", "TestRename", "TestRenameOverwriteDest", "TestRenameFailed", "TestRenameNotExisting", "TestRenameToDirFailed", "TestRenameCaseDifference"}},
			{Name: "os_unix_test.go", Declarations: []string{"checkUidGid", "TestChown", "TestFileChown", "TestLchown"}},
			{Name: "stat_test.go", Declarations: []string{"testStatAndLstatParams", "testStatAndLstat", "testIsDir", "testIsSymlink", "testIsFile", "testDirStats", "testFileStats", "testSymlinkStats", "testSymlinkSameFile", "testSymlinkSameFileOpen", "TestDirAndSymlinkStats", "TestFileAndSymlinkStats", "TestSymlinkWithTrailingSlash", "TestClosedStat", "TestStatNotExist"}},
			{Name: "env_test.go"}, {Name: "env_unix_test.go"}, {Name: "error_test.go"},
			{Name: "read_test.go", Declarations: []string{"checkNamedSize", "TestReadFile", "TestWriteFile", "TestReadOnlyWriteFile"}},
			{Name: "tempfile_test.go", Declarations: []string{"TestCreateTemp", "TestCreateTempPattern", "TestMkdirTemp", "TestMkdirTempBadDir"}},
			{Name: "removeall_test.go", Declarations: []string{"testRequiresPermissions", "TestRemoveAll", "TestRemoveAllLarge", "TestRemoveAllLongPath", "TestRemoveAllDot", "TestRemoveAllDotDot", "TestRemoveReadOnlyDir", "TestRemoveAllButReadOnlyAndPathError", "TestRemoveUnreadableDir", "TestRemoveAllWithMoreErrorThanReqSize", "TestRemoveAllTrailingSlash", "TestRemoveAllSymlinkRemovalFailure"}},
			{Name: "path_test.go", Declarations: []string{"TestMkdirAll", "TestMkdirAllAbsPath", "TestMkdirAllWithSymlink"}},
			{Name: "root_test.go", Declarations: []string{"testMaybeRooted", "makefs", "hasLink", "rootTest", "rootTest.run", "rootTestCases", "errEndsTest", "TestRootMkdirAll", "TestRootMkdir", "TestRootOpen_File", "TestRootOpen_Directory", "TestRootCreate", "TestRootOpenRoot", "TestRootStat", "TestRootLstat", "TestRootReadlink", "TestRootRemoveFile", "TestRootRemoveDirectory", "TestRootNonPermissionMode", "TestRootUseAfterClose", "TestRootSymlinkToRoot", "TestOpenInRoot", "TestRootRemoveDot", "TestRootWriteReadFile", "TestRootName", "TestRootConcurrentClose"}},
		},
		"net":      {{Name: "interface_test.go", Declarations: []string{"TestInterfaces"}}, {Name: "ip_test.go"}, {Name: "mac_test.go"}, {Name: "pipe_test.go"}, {Name: "tcpsock_test.go", Declarations: []string{"TestTCPConcurrentAccept"}}},
		"net/http": {{Name: "header_test.go", Declarations: []string{"headerWriteTests", "TestHeaderWrite", "parseTimeTests", "TestParseTime", "TestNilHeaderClone"}}, {Name: "sniff_test.go", Declarations: []string{"sniffTests", "TestDetectContentType"}}},
	}
	names := make([]string, 0, len(selections))
	for pkg := range selections {
		names = append(names, pkg)
	}
	selected := testShard(t, names)
	for pkg, files := range selections {
		if !selected[pkg] {
			continue
		}
		t.Run(pkg, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(runtime.GOROOT(), "src", filepath.FromSlash(pkg))
			fixture := t.TempDir()
			module := "example.com/upstream"
			if pkg == "reflect" {
				module = "reflect_test"
				// Keep the original package name and import path for reflection.
				writeFile(t, filepath.Join(fixture, "000_package.go"), "package reflect_test\n")
			}
			writeFile(t, filepath.Join(fixture, "go.mod"), "module "+module+"\n\ngo 1.27\n")
			if pkg == "sync" || pkg == "crypto/x509" || pkg == "os" || pkg == "internal/godebug" {
				module := "example.com/upstream"
				if pkg == "sync" {
					module = "sync_test"
				}
				dependencies := "require example.com/testenv v0.0.0\nreplace example.com/testenv => ./testenv\n"
				if pkg == "internal/godebug" {
					dependencies += "\nrequire example.com/bisect v0.0.0\nreplace example.com/bisect => ./bisect\n"
				}
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
				if pkg == "crypto/md5" {
					source = strings.Replace(source, "package md5\n", "package md5_test\nimport . \"crypto/md5\"\n", 1)
				}
				if pkg == "crypto/subtle" {
					source = strings.Replace(source, "package subtle\n", "package subtle_test\nimport . \"crypto/subtle\"\n", 1)
				}
				if pkg == "os/signal" {
					source = strings.Replace(source, "package signal\n", `package signal_test
import . "os/signal"
`, 1)
				}
				if pkg == "net" {
					source = strings.Replace(source, "package net\n", "package net_test\nimport . \"net\"\n", 1)
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
					source = strings.ReplaceAll(source, `"internal/testenv"`, `"example.com/testenv"`)
				}
				if pkg == "go/parser" || pkg == "html/template" {
					name := filepath.Base(pkg)
					wrapper := "package " + name + "_test\n"
					if strings.HasSuffix(file.Name, "_test.go") {
						wrapper += "import . " + strconv.Quote(pkg) + "\n"
					}
					source = strings.Replace(source, "package "+name+"\n", wrapper, 1)
				}
				writeFile(t, filepath.Join(fixture, file.Name), source)
			}
			if pkg == "sync" || pkg == "crypto/x509" || pkg == "os" || pkg == "internal/godebug" {
				names := []string{"Executable", "MustHaveExec"}
				if pkg == "internal/godebug" {
					names = []string{"MustHaveGoRun", "MustHaveGoBuild", "goBuildError", "GoToolPath", "GoTool", "goToolPath", "GOROOT"}
				}
				if pkg == "crypto/x509" {
					names = []string{"Builder", "SkipFlaky"}
				} else if pkg == "os" {
					names = []string{"MustHaveSymlink", "MustHaveExec", "Executable"}
				}
				source := selectedUpstreamSource(t, "../overlay/std/internal/testenv/testenv.go", names)
				if pkg == "internal/godebug" {
					source = strings.ReplaceAll(source, "runtime.GOROOT()", strconv.Quote(runtime.GOROOT()))
				}
				if err := os.MkdirAll(filepath.Join(fixture, "testenv"), 0755); err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(fixture, "testenv/go.mod"), "module example.com/testenv\n\ngo 1.27\n")
				writeFile(t, filepath.Join(fixture, "testenv/testenv.go"), strings.TrimPrefix(source, "//go:build dart\n"))
				if pkg == "os" {
					writeFile(t, filepath.Join(fixture, "testenv/exec.go"), selectedUpstreamSource(t, "../overlay/std/internal/testenv/exec.go", nil))
					writeFile(t, filepath.Join(fixture, "testenv/syscall.go"), selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/internal/testenv/testenv.go"), []string{"SyscallIsNotSupported"}))
					writeFile(t, filepath.Join(fixture, "testenv/syscall_unix.go"), selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/internal/testenv/testenv_unix.go"), []string{"syscallIsNotSupported"}))
				}
			}

			if pkg == "net" {
				writeFile(t, filepath.Join(fixture, "go.mod"), "module example.com/upstream\n\ngo 1.27\n\nrequire golang.org/x/net/nettest v0.0.0\nreplace golang.org/x/net/nettest => ./nettest\n")
				nettest := filepath.Join(runtime.GOROOT(), "src", "vendor", "golang.org", "x", "net", "nettest")
				if err := os.Mkdir(filepath.Join(fixture, "nettest"), 0755); err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(fixture, "nettest/go.mod"), "module golang.org/x/net/nettest\n\ngo 1.27\n")
				writeFile(t, filepath.Join(fixture, "nettest/conntest.go"), selectedUpstreamSource(t, filepath.Join(nettest, "conntest.go"), nil))
				writeFile(t, filepath.Join(fixture, "nettest/constants.go"), selectedUpstreamSource(t, filepath.Join(nettest, "nettest.go"), []string{"aLongTimeAgo", "neverTimeout"}))
				for target, origin := range map[string]string{"LICENSE": filepath.Join(runtime.GOROOT(), "LICENSE"), "nettest/LICENSE": filepath.Join(nettest, "..", "LICENSE")} {
					license, err := os.ReadFile(origin)
					if err != nil {
						t.Fatal(err)
					}
					writeFile(t, filepath.Join(fixture, target), string(license))
				}
				source := selectedUpstreamSource(t, "../overlay/std/net/addr.go", []string{"networkNumberAndMask"})
				source = strings.Replace(source, "package net\n", "package net_test\nimport . \"net\"\n", 1)
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
				text := strings.ReplaceAll(string(data), `"internal/bisect"`, `"example.com/bisect"`)
				text = strings.ReplaceAll(text, `"internal/godebugs"`, `"example.com/upstream/internal/godebugs"`)
				writeFile(t, filepath.Join(fixture, "godebug.go"), text)
				if err := os.MkdirAll(filepath.Join(fixture, "internal/godebugs"), 0755); err != nil {
					t.Fatal(err)
				}
				table, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "src/internal/godebugs/table.go"))
				if err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(fixture, "internal/godebugs/table.go"), string(table))
				if err := os.MkdirAll(filepath.Join(fixture, "bisect"), 0755); err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(fixture, "bisect/go.mod"), "module example.com/bisect\n\ngo 1.27\n")
				bisectDir := filepath.Join(runtime.GOROOT(), "src/internal/bisect")
				loaded, err := load.Load(bisectDir, ".")
				if err != nil {
					t.Fatal(err)
				}
				data, ok := loaded.Sources[filepath.Join(bisectDir, "bisect.go")]
				if !ok {
					t.Fatal("missing adapted bisect source")
				}
				writeFile(t, filepath.Join(fixture, "bisect/bisect.go"), string(data))
				license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
				if err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(fixture, "bisect/LICENSE"), string(license))
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
			if pkg == "sync" || pkg == "internal/godebug" || pkg == "os" {
				launcher = prepareUpstreamSelfExec(t, out, stats.Binaries[0].Entry)
			}
			args := []string{"-test.v", "-test.short=" + strconv.FormatBool(testing.Short()), "-test.run=" + filter, "-test.skip=" + upstreamSkip(t, pkg)}
			result := programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			name := pkg + "-selected"
			if pkg == "net/http" {
				name = pkg + "-content-type"
			}
			compareUpstreamCertificates(t, name, pkg, filter, "vm", dir, result.Text, result.Exit, certificates)
			if pkg == "os/signal" || pkg == "crypto/md5" || pkg == "crypto/subtle" || pkg == "reflect" || pkg == "crypto/x509" || pkg == "crypto/tls" || pkg == "runtime" || pkg == "runtime/metrics" || pkg == "fmt" || pkg == "encoding/asn1" || pkg == "go/parser" || pkg == "html/template" || pkg == "context" || pkg == "testing" {
				for _, target := range []string{"aot", "js", "wasm"} {
					t.Run(target, func(t *testing.T) {
						result := runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
						compareUpstreamCertificates(t, name, pkg, filter, target, dir, result.Text, result.Exit, certificates)
					})
				}
			}
			if pkg == "os" || pkg == "net" {
				t.Run("aot", func(t *testing.T) {
					result := runUpstreamTarget(t, "aot", out, stats.Binaries[0].Entry, dir, args)
					compareUpstream(t, name, pkg, filter, "aot", dir, result.Text, result.Exit)
				})
			}
			if pkg == "net" {
				const pipeFilter = "^(TestPipe|TestPipeCloseError)$"
				pipeArgs := slices.Clone(args)
				pipeArgs[2] = "-test.run=" + pipeFilter
				for _, target := range []string{"js", "wasm"} {
					t.Run(target, func(t *testing.T) {
						result := runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, pipeArgs)
						compareUpstream(t, "net-pipe", pkg, pipeFilter, target, dir, result.Text, result.Exit)
					})
				}
			}
			if pkg == "sync" || pkg == "internal/godebug" {
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

// Keep original declaration bytes and source positions. Adapt imports and the wrapper.
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
		out.WriteString(fmt.Sprintf("//line %s:%d\n", filepath.Base(path), fs.Position(start).Line))
		out.Write(source[fs.Position(start).Offset:fs.Position(decl.End()).Offset])
		out.WriteString("\n\n")
	}
	return out.String()
}
