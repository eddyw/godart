package driver_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/parity"
)

// Literal addresses do not call the upstream DNS hook. Reject a changed
// table before removal of that setup.
func literalResolveSource(source, table string) (string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), "resolve_test.go", source, 0)
	if err != nil {
		return "", err
	}
	var rows *ast.CompositeLit
	ast.Inspect(file, func(node ast.Node) bool {
		if spec, ok := node.(*ast.ValueSpec); ok && len(spec.Names) == 1 && spec.Names[0].Name == table && len(spec.Values) == 1 {
			rows, _ = spec.Values[0].(*ast.CompositeLit)
		}
		return true
	})
	if rows == nil || len(rows.Elts) == 0 {
		return "", fmt.Errorf("upstream address table %s is absent or changed", table)
	}
	for _, element := range rows.Elts {
		row, ok := element.(*ast.CompositeLit)
		if !ok || len(row.Elts) != 4 {
			return "", fmt.Errorf("upstream address row changed in %s", table)
		}
		literal, ok := row.Elts[1].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return "", fmt.Errorf("upstream address is not a string literal in %s", table)
		}
		address, err := strconv.Unquote(literal.Value)
		if err != nil {
			return "", err
		}
		host := address
		if table != "resolveIPAddrTests" {
			host, _, err = net.SplitHostPort(address)
			if err != nil {
				return "", err
			}
		}
		host, _, _ = strings.Cut(host, "%")
		if host != "" && net.ParseIP(host) == nil {
			// The IP table also verifies rejection of a TCP network.
			if table != "resolveIPAddrTests" || address != "1.2.3.4:123" ||
				!isUnknownTCPNetwork(row) {
				return "", fmt.Errorf("upstream address %q needs the DNS hook", address)
			}
		}
	}
	if table == "resolveIPAddrTests" {
		// This table resolves literals without an IP socket or privileges.
		const guard = "\tif !testableNetwork(\"ip+nopriv\") {\n\t\tt.Skip(\"ip+nopriv test\")\n\t}\n"
		if strings.Count(source, guard) != 1 {
			return "", fmt.Errorf("upstream IP resolver platform guard changed")
		}
		source = strings.Replace(source, guard, "", 1)
	}
	const setup = "\torigTestHookLookupIP := testHookLookupIP\n\tdefer func() { testHookLookupIP = origTestHookLookupIP }()\n\ttestHookLookupIP = lookupLocalhost\n"
	if strings.Count(source, setup) != 1 {
		return "", fmt.Errorf("upstream resolver hook setup changed")
	}
	return strings.Replace(source, setup, "", 1), nil
}

func isUnknownTCPNetwork(row *ast.CompositeLit) bool {
	network, ok := row.Elts[0].(*ast.BasicLit)
	if !ok || network.Value != `"tcp"` {
		return false
	}
	call, ok := row.Elts[3].(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	name, ok := call.Fun.(*ast.Ident)
	if !ok || name.Name != "UnknownNetworkError" {
		return false
	}
	value, ok := call.Args[0].(*ast.BasicLit)
	return ok && value.Value == `"tcp"`
}

func resolveErrorSource(source string) (string, error) {
	positions := token.NewFileSet()
	file, err := parser.ParseFile(positions, "net.go", source, 0)
	if err != nil {
		return "", err
	}
	var value *ast.ValueSpec
	ast.Inspect(file, func(node ast.Node) bool {
		if spec, ok := node.(*ast.ValueSpec); ok && len(spec.Names) == 1 && spec.Names[0].Name == "errNoSuitableAddress" {
			value = spec
		}
		return true
	})
	if value == nil {
		return "", fmt.Errorf("upstream address error declaration changed")
	}
	return source[:positions.Position(file.Name.End()).Offset] + "\nimport \"errors\"\nvar " +
		source[positions.Position(value.Pos()).Offset:positions.Position(value.End()).Offset] + "\n", nil
}

func TestLiteralResolveSource(t *testing.T) {
	const source = `package net
var addresses = []entry{{"udp", "127.0.0.1:domain", nil, nil}, {"udp6", "[::1%en0]:53", nil, nil}, {"udp", ":0", nil, nil}}
func verify() {
	origTestHookLookupIP := testHookLookupIP
	defer func() { testHookLookupIP = origTestHookLookupIP }()
	testHookLookupIP = lookupLocalhost
	assertAddresses(addresses)
}

`
	adapted, err := literalResolveSource(source, "addresses")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(adapted, "testHookLookupIP") || !strings.Contains(adapted, "assertAddresses(addresses)") {
		t.Fatal("the adapter changed the assertions or retained the hook")
	}
	for _, changed := range []string{
		strings.Replace(source, "127.0.0.1:domain", "localhost:domain", 1),
		strings.Replace(source, "var addresses", "var renamed", 1),
		strings.Replace(source, "lookupLocalhost", "anotherHook", 1),
	} {
		if _, err := literalResolveSource(changed, "addresses"); err == nil {
			t.Fatal("the adapter accepted a changed upstream contract")
		}
	}
}

func TestResolveErrorSource(t *testing.T) {
	const source = "// Original notice.\npackage net_test\nvar (errNoSuitableAddress = errors.New(\"expected\"); unrelated = anotherError{})\n"
	adapted, err := resolveErrorSource(source)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(adapted, "// Original notice.") || !strings.Contains(adapted, `errNoSuitableAddress = errors.New("expected")`) || strings.Contains(adapted, "unrelated") {
		t.Fatal("the adapter changed the source notice or error declaration")
	}
	if _, err := resolveErrorSource(strings.Replace(source, "errNoSuitableAddress", "renamed", 1)); err == nil {
		t.Fatal("the adapter accepted a changed error declaration")
	}
}

func TestIPLiteralResolveSource(t *testing.T) {
	const source = `package net
var resolveIPAddrTests = []entry{{"ip", "127.0.0.1", nil, nil}, {"ip6", "::1%en0", nil, nil}, {"ip", "", nil, nil}, {"tcp", "1.2.3.4:123", nil, UnknownNetworkError("tcp")}}
func verify() {
	if !testableNetwork("ip+nopriv") {
		t.Skip("ip+nopriv test")
	}
	origTestHookLookupIP := testHookLookupIP
	defer func() { testHookLookupIP = origTestHookLookupIP }()
	testHookLookupIP = lookupLocalhost
	assertAddresses(resolveIPAddrTests)
}
`
	adapted, err := literalResolveSource(source, "resolveIPAddrTests")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(adapted, "testableNetwork") || strings.Contains(adapted, "testHookLookupIP") ||
		!strings.Contains(adapted, "assertAddresses(resolveIPAddrTests)") {
		t.Fatal("the adapter changed assertions or retained platform setup")
	}
	for _, changed := range []string{
		strings.Replace(source, "127.0.0.1", "localhost", 1),
		strings.Replace(source, "ip+nopriv", "ip", 1),
		strings.Replace(source, `UnknownNetworkError("tcp")`, `UnknownNetworkError("ip")`, 1),
		strings.Replace(source, `{"tcp",`, `{"ip",`, 1),
	} {
		if _, err := literalResolveSource(changed, "resolveIPAddrTests"); err == nil {
			t.Fatal("the adapter accepted a changed IP resolver contract")
		}
	}
}

func selectedInputLimits(name string, comparison *parity.Comparison) map[string]string {
	if name != "net-selected" {
		return nil
	}
	limits := map[string]string{}
	for _, test := range []string{"TestResolveIPAddr", "TestResolveTCPAddr", "TestResolveUDPAddr"} {
		limits[test] = "The selected source retains the declared table. It omits resolver cases that setupTestData appends in net/main_test.go."
		if i := slices.Index(comparison.Passed, test); i >= 0 {
			comparison.Passed = slices.Delete(comparison.Passed, i, i+1)
			comparison.Partial = append(comparison.Partial, test)
		}
	}
	slices.Sort(comparison.Partial)
	return limits
}

func TestSelectedInputLimits(t *testing.T) {
	comparison := parity.Comparison{
		Passed:   []string{"TestParseIP", "TestResolveIPAddr", "TestResolveTCPAddr"},
		Problems: []string{"TestResolveUDPAddr failed"},
	}
	limits := selectedInputLimits("net-selected", &comparison)
	if len(limits) != 3 || !slices.Equal(comparison.Passed, []string{"TestParseIP"}) ||
		!slices.Equal(comparison.Partial, []string{"TestResolveIPAddr", "TestResolveTCPAddr"}) ||
		!slices.Equal(comparison.Problems, []string{"TestResolveUDPAddr failed"}) {
		t.Fatalf("incorrect partial coverage: %+v, %v", comparison, limits)
	}
	comparison = parity.Comparison{Passed: []string{"TestResolveIPAddr"}}
	if limits := selectedInputLimits("another-selection", &comparison); limits != nil || len(comparison.Partial) != 0 || len(comparison.Passed) != 1 {
		t.Fatalf("unrelated selection changed: %+v, %v", comparison, limits)
	}
}
