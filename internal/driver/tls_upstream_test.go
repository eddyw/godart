package driver_test

import (
	"encoding/json"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/parity"
)

func TestTLSUpstream(t *testing.T) {
	t.Setenv("GOROOT", runtime.GOROOT())
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
	dir := filepath.Join(runtime.GOROOT(), "src/crypto/tls")
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatalf("got %d test binaries", len(stats.Binaries))
	}
	prepareDart(t, out)
	const portable = "^(TestWeakCertCache|TestSignatureSelection|TestLegacyTypeAndHash|TestSupportedSignatureAlgorithms|TestGenerateCertificates|TestRoundUp|TestRemovePadding|TestCertificateSelection|TestBrokenCertificateSkipped|TestDynamicRecordSizingWithStreamCipher|TestDynamicRecordSizingWithCBC|TestDynamicRecordSizingWithAEAD|TestDynamicRecordSizingWithTLSv13|TestHairpinInClose|TestRecordBadVersionTLS13|TestKeyUpdateSpamPostHandshakeTLS13|TestDecodeECHConfigLists|TestSkipBadConfigs|TestPickECHConfigWithInvalidAEADID|TestECHPadding|TestFIPSClientHello|TestHandshakeClientRSARC4|TestHandshakeClientRSAAES128GCM|TestHandshakeClientRSAAES256GCM|TestHandshakeClientECDHERSAAES|TestHandshakeClientECDHEECDSAAES|TestHandshakeClientECDHEECDSAAESGCM|TestHandshakeClientAES256GCMSHA384|TestHandshakeClientAES128CBCSHA256|TestHandshakeClientECDHERSAAES128CBCSHA256|TestHandshakeClientECDHEECDSAAES128CBCSHA256|TestHandshakeClientX25519|TestHandshakeClientP256|TestHandshakeClientHelloRetryRequest|TestHandshakeClientECDHERSAChaCha20|TestHandshakeClientECDHEECDSAChaCha20|TestHandshakeClientAES128SHA256|TestHandshakeClientAES256SHA384|TestHandshakeClientCHACHA20SHA256|TestHandshakeClientRSATLS13|TestHandshakeClientECDSATLS13|TestHandshakeClientEd25519|TestHandshakeClientCertEd25519|TestHandshakeClientCertRSA|TestHandshakeClientCertECDSA|TestHandshakeClientCertRSAPSS|TestHandshakeClientCertRSAPKCS1v15|TestClientKeyUpdate|TestResumption|TestLRUClientSessionCache|TestKeyLogTLS12|TestKeyLogTLS13|TestHandshakeClientALPNMatch|TestServerSelectingUnconfiguredApplicationProtocol|TestHandshakClientSCTs|TestHandshakeClientSCTs|TestRenegotiationRejected|TestRenegotiateOnce|TestRenegotiateTwice|TestRenegotiateTwiceRejected|TestHandshakeClientExportKeyingMaterial|TestHostnameInSNI|TestServerSelectingUnconfiguredCipherSuite|TestVerifyConnection|TestVerifyPeerCertificate|TestFailedWrite|TestBuffering|TestAlertFlushing|TestHandshakeRace|TestGetClientCertificate|TestRSAPSSKeyError|TestCloseClientConnectionOnIdleServer|TestDowngradeCanary|TestResumptionKeepsOCSPAndSCT|TestClientHandshakeContextCancellation|TestTLS13OnlyClientHelloCipherSuite|TestHandshakeRSATooBig|TestTLS13ECHRejectionCallbacks|TestECHTLS12Server|TestMarshalUnmarshal|TestFuzz|TestRejectEmptySCTList|TestRejectEmptySCT|TestRejectDuplicateExtensions|TestECHRemoveOuterPSK|TestSimpleError|TestRejectBadProtocolVersion|TestNoSuiteOverlap|TestNoCompressionOverlap|TestNoRC4ByDefault|TestRejectSNIWithTrailingDot|TestDontSelectECDSAWithRSAKey|TestDontSelectRSAWithECDSAKey|TestRenegotiationExtension|TestTLS12OnlyCipherSuites|TestTLSPointFormats|TestAlertForwarding|TestClose|TestVersion|TestCipherSuitePreference|TestSCTHandshake|TestCrossVersionResume|TestHandshakeServerRSARC4|TestHandshakeServerRSA3DES|TestHandshakeServerRSAAES|TestHandshakeServerAESGCM|TestHandshakeServerAES256GCMSHA384|TestHandshakeServerAES128SHA256|TestHandshakeServerAES256SHA384|TestHandshakeServerCHACHA20SHA256|TestHandshakeServerECDHEECDSAAES|TestHandshakeServerX25519|TestHandshakeServerP256|TestHandshakeServerHelloRetryRequest|TestHandshakeServerKeySharePreference|TestHandshakeServerALPN|TestHandshakeServerALPNNoMatch|TestHandshakeServerALPNNotConfigured|TestHandshakeServerALPNFallback|TestHandshakeServerSNI|TestHandshakeServerSNIGetCertificate|TestHandshakeServerSNIGetCertificateNotFound|TestHandshakeServerGetCertificateExtensions|TestHandshakeServerSNIGetCertificateError|TestHandshakeServerEmptyCertificates|TestServerResumption|TestServerResumptionDisabled|TestFallbackSCSV|TestHandshakeServerExportKeyingMaterial|TestHandshakeServerRSAPKCS1v15|TestHandshakeServerRSAPSS|TestHandshakeServerEd25519|TestClientAuth|TestSNIGivenOnFailure|TestGetConfigForClient|TestCloseServerConnectionOnIdleClient|TestCloneHash|TestKeyTooSmallForRSAPSS|TestMultipleCertificates|TestAESCipherReordering|TestAESCipherReorderingTLS13|TestServerHandshakeContextCancellation|TestHandshakeContextHierarchy|TestHandshakeChainExpiryResumption|TestHandshakeGetConfigForClientDifferentClientCAs|TestHandshakeChangeRootCAsResumption|TestServerHelloTrailingMessage|TestClientHelloTrailingMessage|TestDoubleClientHelloHRR|TestMultipleKeyUpdate|TestACVPVectors|TestTrafficKey|TestSplitPreMasterSecret|TestKeysFromPreMasterSecret|TestQUICConnection|TestQUICVersions|TestQUICSessionResumption|TestQUICFragmentaryData|TestQUICPostHandshakeClientAuthentication|TestQUICPostHandshakeKeyUpdate|TestQUICPostHandshakeMessageTooLarge|TestQUICHandshakeError|TestQUICECHKeyError|TestQUICConnectionState|TestQUICStartContextPropagation|TestQUICClientHelloInfoConn|TestQUICContextCancelation|TestQUICDelayedTransportParameters|TestQUICEmptyTransportParameters|TestQUICCanceledWaitingForData|TestQUICCanceledWaitingForTransportParams|TestQUICEarlyData|TestQUICEarlyDataDeclined|TestX509KeyPair|TestDialTimeout|TestDeadlineOnWrite|TestDialer|TestConnReadNonzeroAndEOF|TestTLSUniqueMatches|TestConnCloseBreakingWrite|TestConnCloseWrite|TestWarningAlertFlood|TestCloneFuncFields|TestCloneNonFuncFields|TestCloneNilConfig|TestConnectionStateMarshal|TestConnectionState|TestBuildNameToCertificate_doesntModifyCertificates|TestClientHelloInfo_SupportsCertificate|TestCipherSuites|TestVersionName|TestPKCS1OnlyCert|TestVerifyCertificates|TestHandshakeMLKEM|TestSupportedSignatureAlgorithmsMLDSAGating|TestHandshakeMLDSA|TestX509KeyPairPopulateCertificate|TestEarlyLargeCertMsg|TestLargeCertMsg|TestECH|TestMessageSigner|TestInvalidHandshakeSignature|TestKeyLogWriterErr|TestLocalCertificate|TestLocalCertificateResumption)$"
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			filter := portable
			var exclusions map[string]string
			if target == "aot" && !testing.Short() {
				filter = portable[:len(portable)-2] + "|TestFIPSCertAlgs)$"
			} else if target == "aot" {
				exclusions = map[string]string{"TestFIPSCertAlgs": "RSA setup can exceed the short-mode command limit. Full AOT comparisons execute the original matrix."}
			} else {
				exclusions = map[string]string{"TestFIPSCertAlgs": "The exhaustive certificate matrix exceeds the unchanged VM command limit. Full VM parity remains unverified."}
			}
			filters := []string{filter}
			if target == "vm" {
				filters = nil
				for names := range slices.Chunk(strings.Split(filter[2:len(filter)-2], "|"), 100) {
					filters = append(filters, "^("+strings.Join(names, "|")+")$")
				}
			}
			for index, filter := range filters {
				t.Run(strconv.Itoa(index), func(t *testing.T) {
					args := upstreamTestArgs("-test.run=" + filter)
					var result programResult
					name := "tls-selected"
					if target == "vm" {
						name += "/vm-" + strconv.Itoa(index)
						result = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
					} else {
						result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
					}
					compareUpstreamInventory(t, name, "crypto/tls", filter, target, dir, result.Text, result.Exit, inventory, nil, nil, exclusions, true)
				})
			}
		})
	}
}
