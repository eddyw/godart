/// The fetch client of the Dart VM.
///
/// The VM holds a socket, thus net/http rides the transport of net.dart there
/// and no code below ever runs. The 3 functions exist because the compiler
/// emits the call of each 1 for both targets, and the choice of the transport
/// happens at run time. Read decision 251 and fetch_web.dart.
library;

import 'slice.dart';
import 'string.dart';

final GoString _noFetch = GoString.fromString('operation not supported');

(int, int, Slice<GoString>, GoString, GoString) $fetchStart(
  GoString method,
  GoString url,
  Slice<GoString> header,
  Slice<int> body,
  int deadline,
) => (-1, 0, Slice<GoString>.nil(), GoString.empty, _noFetch);

(int, GoString) $fetchRead(int handle, Slice<int> p) => (0, _noFetch);

GoString $fetchClose(int handle) => _noFetch;
