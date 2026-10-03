// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:io';

import 'goint.dart';
import 'sched.dart';
import 'slice.dart';
import 'string.dart';
import 'unix_datagram.dart';

final _unixDatagrams = <UnixDatagram?>[];
final _empty = GoString.empty;
final _closed = GoString.fromString('use of closed network connection');

UnixDatagram _at(int handle) {
  final value = _unixDatagrams[handle];
  if (value == null) throw UnixDatagramClosed();
  return value;
}

(GoString, int, bool) _cause(Object error) {
  if (error is UnixDatagramClosed) return (_closed, 0, false);
  if (error is UnixDatagramExpired) {
    return (GoString.fromString('i/o timeout'), 0, true);
  }
  final cause = error as SocketException;
  return (
    GoString.fromString(cause.message),
    cause.osError?.errorCode ?? 0,
    false,
  );
}

(int, GoString, GoString, GoString, int) $unixgramOpen(
  GoString local,
  GoString remote,
  bool bind,
  bool connect,
) {
  try {
    final value = UnixDatagram.open(
      bind ? local : null,
      connect ? remote : null,
    );
    var handle = _unixDatagrams.indexOf(null);
    if (handle < 0) {
      handle = _unixDatagrams.length;
      _unixDatagrams.add(value);
    } else {
      _unixDatagrams[handle] = value;
    }
    return (handle, value.local, value.remote ?? _empty, _empty, 0);
  } catch (error) {
    final (cause, code, _) = _cause(error);
    return (-1, _empty, _empty, cause, code);
  }
}

(int, int, int, GoString, GoString, int, bool) $unixgramRead(
  int handle,
  Slice<int> data,
  Slice<int> control,
) {
  try {
    final socket = _at(handle);
    $await(
      socket
          .read(
            data.array,
            data.off,
            data.off + data.len,
            control.array,
            control.off,
            control.off + control.len,
          )
          .then(
            (result) => (
              $gi(result.$1),
              $gi(result.$2),
              $gi(result.$3),
              result.$4,
              _empty,
              $gi(0),
              false,
            ),
          ),
      (error) {
        final (cause, code, expired) = _cause(error);
        return ($gi(0), $gi(0), $gi(0), _empty, cause, $gi(code), expired);
      },
    );
    return (0, 0, 0, _empty, _empty, 0, false);
  } on UnixDatagramClosed {
    return (0, 0, 0, _empty, _closed, 0, false);
  }
}

(int, int, GoString, int, bool) $unixgramWrite(
  int handle,
  Slice<int> data,
  Slice<int> control,
  GoString destination,
  bool sendTo,
) {
  try {
    final socket = _at(handle);
    $await(
      socket
          .write(
            data.array,
            data.off,
            data.off + data.len,
            control.array,
            control.off,
            control.off + control.len,
            sendTo ? destination : null,
          )
          .then((n) => ($gi(n), $gi(control.len), _empty, $gi(0), false)),
      (error) {
        final (cause, code, expired) = _cause(error);
        return ($gi(0), $gi(0), cause, $gi(code), expired);
      },
    );
    return (0, 0, _empty, 0, false);
  } on UnixDatagramClosed {
    return (0, 0, _closed, 0, false);
  }
}

(GoString, int) $unixgramClose(int handle) {
  try {
    final socket = _at(handle);
    _unixDatagrams[handle] = null;
    socket.close();
    return (_empty, 0);
  } catch (error) {
    final (cause, code, _) = _cause(error);
    return (cause, code);
  }
}

(GoString, int) $unixgramDeadline(
  int handle,
  int timeout,
  int mode,
  bool clear,
) {
  try {
    _at(handle).setDeadline(timeout, mode, clear);
    return (_empty, 0);
  } catch (error) {
    final (cause, code, _) = _cause(error);
    return (cause, code);
  }
}

(GoString, int) $unixgramBuffer(int handle, bool read, int size) {
  try {
    _at(handle).setBuffer(read, size);
    return (_empty, 0);
  } catch (error) {
    final (cause, code, _) = _cause(error);
    return (cause, code);
  }
}

(GoString, int) $unixgramShutdown(int handle, int direction) {
  try {
    _at(handle).shutdown(direction);
    return (_empty, 0);
  } catch (error) {
    final (cause, code, _) = _cause(error);
    return (cause, code);
  }
}

int $syscallUnixSendto(
  int descriptor,
  Slice<int> data,
  int flags,
  GoString destination,
  bool hasDestination,
) => unixDatagramSendto(
  descriptor,
  data.array,
  data.off,
  data.off + data.len,
  flags,
  hasDestination ? destination : null,
);
