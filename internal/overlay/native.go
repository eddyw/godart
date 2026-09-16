package overlay

import (
	"strings"
	"sync"
)

// native lists each standard library package that the compiler reads from the
// real Go source. Such a package needs no overlay, because it already stays
// inside the subset.
//
// A package enters this list when stage 2 accepts each file of it, and when
// each package that it imports also stays in the surface. A directory of the
// same path inside std then changes 1 file of the package by its base name: a
// `.go` file replaces the whole file, and a `.patch` file changes the named
// lines of the real Go source. A diff is the better form for a small change,
// because it names the lines that differ and it ages with no copy.
var native = []string{
	"fmt",
	"bufio",
	"bytes",
	"cmp",
	"container/list",
	"encoding/base64",
	"internal/byteorder",
	"internal/oserror",
	"internal/nettrace",
	"io/fs",
	"context",
	"io",
	"path",
	"math",
	"math/bits",
	"math/cmplx",
	"math/rand",
	"math/rand/v2",
	"internal/chacha8rand",
	"unicode",
	"container/heap",
	"container/ring",
	"encoding",
	"encoding/ascii85",
	"encoding/csv",
	"encoding/hex",
	"hash",
	"internal/testhash",
	"hash/adler32",
	"hash/crc32",
	"hash/fnv",
	"net/netip",
	"regexp",
	"regexp/syntax",
	"net/url",
	"compress/flate",
	"compress/lzw",
	"compress/bzip2",
	"archive/zip",
	"archive/tar",
	"html",
	"encoding/xml",
	"math/big",
	"html/template",
	"compress/gzip",
	"compress/zlib",
	"testing/iotest",
	"text/tabwriter",
	"text/template/parse",
	"text/scanner",
	"text/template",
	"internal/fmtsort",
	"go/token",
	"go/scanner",
	"go/ast",
	"go/build/constraint",
	"go/parser",
	"go/doc/comment",
	"go/printer",
	"go/format",
	"go/constant",
	"internal/goversion",
	"internal/gover",
	"go/version",
	"internal/types/errors",
	"hash/maphash",
	"go/types",
	"internal/lazyregexp",
	"go/doc",
	"internal/goexperiment",
	"internal/buildcfg",
	"internal/goroot",
	"internal/platform",
	"internal/syslist",
	"internal/dag",
	"go/build",
	"net/textproto",
	"mime",
	"mime/quotedprintable",
	"mime/multipart",
	"net/mail",
	"net/smtp",
	"net/rpc",
	"vendor/golang.org/x/text/transform",
	"vendor/golang.org/x/text/unicode/bidi",
	"vendor/golang.org/x/text/unicode/norm",
	"vendor/golang.org/x/text/secure/bidirule",
	"vendor/golang.org/x/net/idna",
	"vendor/golang.org/x/net/http/httpguts",
	"net/http/cgi",
	"net/rpc/jsonrpc",
	"net/http/cookiejar",
	"net/http/internal/ascii",
	"net/http/internal",
	"net/http/httptrace",
	"net/http/httputil",
	"unicode/utf16",
	"encoding/pem",
	"encoding/asn1",
	"crypto/x509/pkix",
	"uuid",
	"database/sql/internal",
	"database/sql/driver",
	"database/sql",
	"encoding/base32",
	"hash/crc64",
	"io/ioutil",
	"index/suffixarray",
	"testing/fstest",
	"log/internal",
	"log/slog/internal",
	"log/slog/internal/buffer",
	"log/slog",
	"testing/slogtest",
	"debug/dwarf",
	"internal/zstd",
	"internal/saferio",
	"debug/plan9obj",
	"debug/macho",
	"debug/pe",
	"debug/elf",
	"image/color",
	"image/color/palette",
	"image",
	"image/internal/imageutil",
	"image/draw",
	"image/gif",
	"image/jpeg",
	"image/png",
	"crypto",
	"crypto/internal/constanttime",
	"crypto/internal/impl",
	"crypto/internal/fips140deps/byteorder",
	"crypto/internal/fips140deps/cpu",
	"crypto/internal/fips140deps/godebug",
	"crypto/internal/fips140deps/time",
	"crypto/internal/fips140",
	"crypto/internal/fips140/alias",
	"crypto/internal/fips140/sha256",
	"crypto/internal/fips140/sha512",
	"crypto/internal/fips140/sha3",
	"crypto/internal/fips140/hmac",
	"crypto/internal/fips140/check",
	"crypto/fips140",
	"crypto/internal/fips140/subtle",
	"crypto/internal/fips140/drbg",
	"crypto/internal/fips140/aes",
	"crypto/internal/fips140/aes/gcm",
	"crypto/internal/fips140only",
	"crypto/internal/boring/sig",
	"crypto/internal/boring",
	"crypto/cipher",
	"crypto/aes",
	"crypto/des",
	"crypto/rc4",
	"crypto/sha3",
	"crypto/internal/fips140hash",
	"crypto/internal/fips140/hkdf",
	"crypto/internal/fips140/pbkdf2",
	"crypto/hkdf",
	"crypto/pbkdf2",
	"crypto/internal/fips140/nistec/fiat",
	"crypto/internal/fips140/nistec",
	"crypto/elliptic",
	"crypto/internal/randutil",
	"crypto/internal/rand",
	"crypto/internal/fips140/edwards25519/field",
	"crypto/internal/fips140/ecdh",
	"crypto/ecdh",
	"crypto/internal/fips140/edwards25519",
	"crypto/internal/fips140/ed25519",
	"crypto/ed25519",
	"crypto/internal/fips140/bigmod",
	"crypto/internal/boring/bbig",
	"vendor/golang.org/x/crypto/cryptobyte/asn1",
	"vendor/golang.org/x/crypto/cryptobyte",
	"crypto/internal/fips140/ecdsa",
	"crypto/ecdsa",
	"crypto/internal/fips140/rsa",
	"crypto/rsa",
	"crypto/dsa",
	"crypto/internal/fips140/mlkem",
	"crypto/mlkem",
	"crypto/mlkem/mlkemtest",
	"crypto/internal/fips140/mldsa",
	"crypto/mldsa",
	"vendor/golang.org/x/crypto/internal/alias",
	"vendor/golang.org/x/crypto/chacha20",
	"vendor/golang.org/x/crypto/internal/poly1305",
	"vendor/golang.org/x/crypto/chacha20poly1305",
	"crypto/hpke",
	"testing/cryptotest",
	"internal/testlog",
	"debug/gosym",
	// sort reads the real algorithm of Go, because 4 tests measure the count
	// of the swaps and of the comparisons. slice.go holds the 1 patch.
	"sort",
	// slices reads the real source, because 4 rules of it hold a value that
	// the hand written form gave in another way: the clear of the tail of a
	// Delete, the bound of an Insert, the nil result of an empty Concat, and
	// the overlap test. slices.go holds the 1 patch.
	"slices",
	"encoding/json/internal",
	"encoding/json/internal/jsonflags",
	"encoding/json/internal/jsonopts",
	"encoding/json/internal/jsonwire",
	"encoding/json/jsontext",
	"encoding/json/v2",
	"encoding/json",
	// jsontest holds the corpus of the real test files of encoding/json and of
	// encoding/json/v2. It reads internal/zstd and embed, and both stand above.
	"encoding/json/internal/jsontest",
}

// NativePaths gives the standard library packages that the compiler reads from
// the real Go source.
func NativePaths() []string {
	out := make([]string, len(native))
	copy(out, native)
	return out
}

// Surface gives each standard library package that the compiler writes Dart
// code for. It joins the overlays and the native packages.
func Surface() []string {
	return append(Paths(), native...)
}

// Outside reports whether a path names a standard library package that the
// curated surface does not hold. The compiler writes no Dart code for such a
// package, and the verifier rejects an import of it.
//
// The first element of a module path always holds a dot, and a standard path
// never does. A path with the suffix _test names an external test package. The
// go command reserves that suffix, thus such a package belongs to the build.
func Outside(pkgPath string) bool {
	if strings.HasSuffix(pkgPath, "_test") {
		return false
	}
	first, _, _ := strings.Cut(pkgPath, "/")
	if strings.Contains(first, ".") {
		return false
	}
	return !surfaceSet()[pkgPath]
}

// surfaceSet holds each package of the surface. A walk of the embedded files
// for each call of Outside would cost too much.
var surfaceSet = sync.OnceValue(func() map[string]bool {
	out := map[string]bool{}
	for _, p := range Surface() {
		out[p] = true
	}
	return out
})
