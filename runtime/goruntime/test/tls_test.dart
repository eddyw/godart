import 'dart:io';

import 'package:goruntime/goruntime.dart';
import 'package:test/test.dart';

/// A certificate of a test, for the host 127.0.0.1 and example.com. It comes
/// from net/http/internal/testcert of the Go repository, and it expires in
/// 2084.
const String _cert = '''
-----BEGIN CERTIFICATE-----
MIIDSDCCAjCgAwIBAgIQEP/md970HysdBTpuzDOf0DANBgkqhkiG9w0BAQsFADAS
MRAwDgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYw
MDAwWjASMRAwDgYDVQQKEwdBY21lIENvMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEAxcl69ROJdxjN+MJZnbFrYxyQooADCsJ6VDkuMyNQIix/Hk15Nk/u
FyBX1Me++aEpGmY3RIY4fUvELqT/srvAHsTXwVVSttMcY8pcAFmXSqo3x4MuUTG/
jCX3Vftj0r3EM5M8ImY1rzA/jqTTLJg00rD+DmuDABcqQvoXw/RV8w1yTRi5BPoH
DFD/AWTt/YgMvk1l2Yq/xI8VbMUIpjBoGXxWsSevQ5i2s1mk9/yZzu0Ysp1tTlzD
qOPa4ysFjBitdXiwfxjxtv5nXqOCP5rheKO0sWLk0fetMp1OV5JSJMAJw6c2ZMkl
U2WMqAEpRjdE/vHfIuNg+yGaRRqI07NZRQIDAQABo4GXMIGUMA4GA1UdDwEB/wQE
AwICpDATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1Ud
DgQWBBQR5QIzmacmw78ZI1C4MXw7Q0wJ1jA9BgNVHREENjA0ggtleGFtcGxlLmNv
bYINKi5leGFtcGxlLmNvbYcEfwAAAYcQAAAAAAAAAAAAAAAAAAAAATANBgkqhkiG
9w0BAQsFAAOCAQEACrRNgiioUDzxQftd0fwOa6iRRcPampZRDtuaF68yNHoNWbOu
LUwc05eOWxRq3iABGSk2xg+FXM3DDeW4HhAhCFptq7jbVZ+4Jj6HeJG9mYRatAxR
Y/dEpa0D0EHhDxxVg6UzKOXB355n0IetGE/aWvyTV9SiDs6QsaC57Q9qq1/mitx5
2GFBoapol9L5FxCc77bztzK8CpLujkBi25Vk6GAFbl27opLfpyxkM+rX/T6MXCPO
6/YBacNZ7ff1/57Etg4i5mNA6ubCpuc4Gi9oYqCNNohftr2lkJr7REdDR6OW0lsL
rF7r4gUnKeC7mYIH1zypY7laskopiLFAfe96Kg==
-----END CERTIFICATE-----
''';

const String _key = '''
-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDFyXr1E4l3GM34
wlmdsWtjHJCigAMKwnpUOS4zI1AiLH8eTXk2T+4XIFfUx775oSkaZjdEhjh9S8Qu
pP+yu8AexNfBVVK20xxjylwAWZdKqjfHgy5RMb+MJfdV+2PSvcQzkzwiZjWvMD+O
pNMsmDTSsP4Oa4MAFypC+hfD9FXzDXJNGLkE+gcMUP8BZO39iAy+TWXZir/EjxVs
xQimMGgZfFaxJ69DmLazWaT3/JnO7RiynW1OXMOo49rjKwWMGK11eLB/GPG2/mde
o4I/muF4o7SxYuTR960ynU5XklIkwAnDpzZkySVTZYyoASlGN0T+8d8i42D7IZpF
GojTs1lFAgMBAAECggEAIYthUi1lFBDd5gG4Rzlu+BlBIn5JhcqkCqLEBiJIFfOr
/4yuMRrvS3bNzqWt6xJ9MSAC4ZlN/VobRLnxL/QNymoiGYUKCT3Ww8nvPpPzR9OE
sE68TUL9tJw/zZJcRMKwgvrGqSLimfq53MxxkE+kLdOc0v9C8YH8Re26mB5ZcWYa
7YFyZQpKsQYnsmu/05cMbpOQrQWhtmIqRoyn8mG/par2s3NzjtpSE9NINyz26uFc
k/3ovFJQIHkUmTS7KHD3BgY5vuCqP98HramYnOysJ0WoYgvSDNCWw3037s5CCwJT
gCKuM+Ow6liFrj83RrdKBpm5QUGjfNpYP31o+QNP4QKBgQDSrUQ2XdgtAnibAV7u
7kbxOxro0EhIKso0Y/6LbDQgcXgxLqltkmeqZgG8nC3Z793lhlSasz2snhzzooV5
5fTy1y8ikXqjhG0nNkInFyOhsI0auE28CFoDowaQd+5cmCatpN4Grqo5PNRXxm1w
HktfPEgoP11NNCFHvvN5fEKbbQKBgQDwVlOaV20IvW3IPq7cXZyiyabouFF9eTRo
VJka1Uv+JtyvL2P0NKkjYHOdN8gRblWqxQtJoTNk020rVA4UP1heiXALy50gvj/p
hMcybPTLYSPOhAGx838KIcvGR5oskP1aUCmFbFQzGELxhJ9diVVjxUtbG2DuwPKd
tD9TLxT2OQKBgQCcdlHSjp+dzdgERmBa0ludjGfPv9/uuNizUBAbO6D690psPFtY
JQMYaemgSd1DngEOFVWADt4e9M5Lose+YCoqr+UxpxmNlyv5kzJOFcFAs/4XeglB
PHKdgNW/NVKxMc6H54l9LPr+x05sYdGlEtqnP/3W5jhEvhJ5Vjc8YiyVgQKBgQCl
zwjyrGo+42GACy7cPYE5FeIfIDqoVByB9guC5bD98JXEDu/opQQjsgFRcBCJZhOY
M0UsURiB8ROaFu13rpQq9KrmmF0ZH+g8FSzQbzcbsTLg4VXCDXmR5esOKowFPypr
Sm667BfTAGP++D5ya7MLmCv6+RKQ5XD8uEQQAaV2kQKBgAD8qeJuWIXZT0VKkQrn
nIhgtzGERF/6sZdQGW2LxTbUDWG74AfFkkEbeBfwEkCZXY/xmnYqYABhvlSex8jU
supU6Eea21esIxIub2zv/Np0ojUb6rlqTPS4Ox1E27D787EJ3VOXpriSD10vyNnZ
jel6uj2FOP9g54s+GzlSVg/T
-----END PRIVATE KEY-----
''';

/// The frame and the machine below hold the shape that stage 5 writes for a
/// function that can block. The machine dials a TLS server, writes 4 bytes,
/// reads the answer, and closes the connection.
final class _TlsFrame implements Frame {
  final int port;
  final List<String> log;
  final Completer<void> done;
  int s = 0;
  bool c = false;
  Frame? r;
  int handle = -1;
  GoString proto = GoString.empty;
  GoString cause = GoString.empty;
  int n = 0;
  Slice<int> buf = Slice.of(Uint8List(8));

  _TlsFrame(this.port, this.log, this.done);

  @override
  Object? resume() {
    _talk(port, log, done, this);
    return null;
  }
}

void _talk(int port, List<String> log, Completer<void> done, [_TlsFrame? f]) {
  int s = 0;
  bool c = false;
  Frame? r;
  int handle = -1;
  GoString proto = GoString.empty;
  GoString cause = GoString.empty;
  int n = 0;
  Slice<int> buf = Slice.of(Uint8List(8));
  if (f != null) {
    s = f.s;
    c = f.c;
    r = f.r;
    handle = f.handle;
    proto = f.proto;
    cause = f.cause;
    n = f.n;
    buf = f.buf;
  }
  machine:
  while (true) {
    switch (s) {
      case 0:
        (handle, proto, cause) = $dialTLS(
          GoString.fromString('127.0.0.1'),
          port,
          true,
          GoString.empty,
          Slice<GoString>.of(<GoString>[GoString.fromString('gart/1')]),
        );
        s = 1;
        continue machine;
      case 1:
        if (c) {
          c = false;
          (handle, proto, cause) = r!.resume() as (int, GoString, GoString);
        }
        if ($blocked != null) break machine;
        log.add(
          'dial proto=${proto.toDartString()} cause=${cause.toDartString()}',
        );
        (n, cause, _) = $connWrite(
          handle,
          $stringToBytes(GoString.fromString('ping')),
        );
        s = 2;
        continue machine;
      case 2:
        if (c) {
          c = false;
          (n, cause, _) = r!.resume() as (int, GoString, int);
        }
        if ($blocked != null) break machine;
        log.add('write n=$n cause=${cause.toDartString()}');
        (n, cause, _) = $connRead(handle, buf);
        s = 3;
        continue machine;
      case 3:
        if (c) {
          c = false;
          (n, cause, _) = r!.resume() as (int, GoString, int);
        }
        if ($blocked != null) break machine;
        final String text = String.fromCharCodes(buf.array.take(n));
        log.add('read text=$text cause=${cause.toDartString()}');
        log.add('close cause=${$connClose(handle).toDartString()}');
        done.complete();
        return;
    }
  }
  $block(
    _TlsFrame(port, log, done)
      ..s = s
      ..c = true
      ..r = $blocked
      ..handle = handle
      ..proto = proto
      ..cause = cause
      ..n = n
      ..buf = buf,
  );
}

/// Starts a TLS server that writes each packet back.
Future<SecureServerSocket> _startEcho() async {
  final SecurityContext context = SecurityContext(withTrustedRoots: false);
  context.useCertificateChainBytes(_cert.codeUnits);
  context.usePrivateKeyBytes(_key.codeUnits);
  context.setAlpnProtocols(<String>['gart/1'], true);
  final SecureServerSocket server = await SecureServerSocket.bind(
    InternetAddress.loopbackIPv4,
    0,
    context,
  );
  server.listen((SecureSocket socket) {
    socket.listen(socket.add, onDone: socket.destroy);
  });
  return server;
}

void main() {
  group('the TLS access', () {
    test('a handshake, a write, and a read pass through a frame', () async {
      final SecureServerSocket server = await _startEcho();
      final List<String> log = <String>[];
      final Completer<void> done = Completer<void>();
      $go(() => _talk(server.port, log, done));
      $schedule();
      await done.future;
      await server.close();
      expect(log, <String>[
        'dial proto=gart/1 cause=',
        'write n=4 cause=',
        'read text=ping cause=',
        'close cause=',
      ]);
    });

    test('a dial to a port with no TLS gives a cause', () async {
      final ServerSocket plain = await ServerSocket.bind(
        InternetAddress.loopbackIPv4,
        0,
      );
      plain.listen((Socket socket) => socket.destroy());
      final List<String> log = <String>[];
      final Completer<void> done = Completer<void>();
      $go(() => _talk(plain.port, log, done));
      $schedule();
      await done.future;
      await plain.close();
      expect(log[0], startsWith('dial proto= cause='));
      expect(log[0], isNot('dial proto= cause='));
    });
  });
}
