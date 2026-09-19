/// The web form of http2.dart.
///
/// A web target holds no socket, thus it runs no HTTP/2 session of its own.
/// This file still declares each name of http2.dart, because net/http names
/// each 1. A program that only builds a request would otherwise fail in the
/// Dart compile.
///
/// Each operation gives the cause "operation not supported".
library;

import 'goint.dart';
import 'slice.dart';
import 'string.dart';

/// The cause of each operation that the web target does not hold. Go gives
/// the same text for the errno ENOTSUP.
final GoString _noNet = GoString.fromString('operation not supported');

final GoString _empty = GoString.empty;

Slice<GoString> _noFields() => Slice<GoString>.nil();

(int, GoString) $h2Client(int conn) => (-1, _noNet);

(int, int, Slice<GoString>, GoString) $h2Fetch(
  int session,
  GoString method,
  GoString path,
  GoString authority,
  Slice<GoString> headers,
  Slice<GoInt> body,
  bool hasBody,
) => (-1, 0, _noFields(), _noNet);

(int, GoString) $h2Read(int stream, Slice<GoInt> p) => (0, _noNet);

(int, GoString) $h2Server(int conn) => (-1, _noNet);

(int, GoString, GoString, GoString, Slice<GoString>, GoString) $h2Next(
  int session,
) => (-1, _empty, _empty, _empty, _noFields(), _noNet);

GoString $h2WriteHead(
  int stream,
  int status,
  Slice<GoString> headers,
  bool endStream,
) => _noNet;

(int, GoString) $h2Write(int stream, Slice<GoInt> p) => (0, _noNet);

GoString $h2End(int stream) => _noNet;

GoString $h2Close(int session) => _noNet;
