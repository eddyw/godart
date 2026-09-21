// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import 'dart:ffi';
import 'dart:io';

import 'slice.dart';
import 'string.dart';

typedef _Ref = Pointer<Void>;

final _security = DynamicLibrary.open(
  '/System/Library/Frameworks/Security.framework/Security',
);
final _core = DynamicLibrary.open(
  '/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation',
);
final _libc = DynamicLibrary.process();
final _malloc = _libc.lookupFunction<_Ref Function(Size), _Ref Function(int)>(
  'malloc',
);
final _free = _libc.lookupFunction<Void Function(_Ref), void Function(_Ref)>(
  'free',
);
final _release = _core.lookupFunction<Void Function(_Ref), void Function(_Ref)>(
  'CFRelease',
);
final _arrayCreate = _core
    .lookupFunction<
      _Ref Function(_Ref, IntPtr, _Ref),
      _Ref Function(_Ref, int, _Ref)
    >('CFArrayCreateMutable');
final _arrayAppend = _core
    .lookupFunction<Void Function(_Ref, _Ref), void Function(_Ref, _Ref)>(
      'CFArrayAppendValue',
    );
final _arrayCount = _core
    .lookupFunction<IntPtr Function(_Ref), int Function(_Ref)>(
      'CFArrayGetCount',
    );
final _arrayAt = _core
    .lookupFunction<_Ref Function(_Ref, IntPtr), _Ref Function(_Ref, int)>(
      'CFArrayGetValueAtIndex',
    );
final _dataCreate = _core
    .lookupFunction<
      _Ref Function(_Ref, Pointer<Uint8>, IntPtr),
      _Ref Function(_Ref, Pointer<Uint8>, int)
    >('CFDataCreate');
final _dataLength = _core
    .lookupFunction<IntPtr Function(_Ref), int Function(_Ref)>(
      'CFDataGetLength',
    );
final _dataBytes = _core
    .lookupFunction<
      Pointer<Uint8> Function(_Ref),
      Pointer<Uint8> Function(_Ref)
    >('CFDataGetBytePtr');
final _stringCreate = _core
    .lookupFunction<
      _Ref Function(_Ref, Pointer<Uint8>, IntPtr, Uint32, Uint8),
      _Ref Function(_Ref, Pointer<Uint8>, int, int, int)
    >('CFStringCreateWithBytes');
final _stringData = _core
    .lookupFunction<
      _Ref Function(_Ref, _Ref, Uint32, Uint8),
      _Ref Function(_Ref, _Ref, int, int)
    >('CFStringCreateExternalRepresentation');
final _errorDescription = _core
    .lookupFunction<_Ref Function(_Ref), _Ref Function(_Ref)>(
      'CFErrorCopyDescription',
    );
final _errorCode = _core
    .lookupFunction<IntPtr Function(_Ref), int Function(_Ref)>(
      'CFErrorGetCode',
    );
final _dateCreate = _core
    .lookupFunction<_Ref Function(_Ref, Double), _Ref Function(_Ref, double)>(
      'CFDateCreate',
    );
final _certificateCreate = _security
    .lookupFunction<_Ref Function(_Ref, _Ref), _Ref Function(_Ref, _Ref)>(
      'SecCertificateCreateWithData',
    );
final _policyCreate = _security
    .lookupFunction<_Ref Function(Uint8, _Ref), _Ref Function(int, _Ref)>(
      'SecPolicyCreateSSL',
    );
final _trustCreate = _security
    .lookupFunction<
      Int32 Function(_Ref, _Ref, Pointer<_Ref>),
      int Function(_Ref, _Ref, Pointer<_Ref>)
    >('SecTrustCreateWithCertificates');
final _trustDate = _security
    .lookupFunction<Int32 Function(_Ref, _Ref), int Function(_Ref, _Ref)>(
      'SecTrustSetVerifyDate',
    );
final _trustEvaluate = _security
    .lookupFunction<
      Bool Function(_Ref, Pointer<_Ref>),
      bool Function(_Ref, Pointer<_Ref>)
    >('SecTrustEvaluateWithError');
final _trustChain = _security
    .lookupFunction<_Ref Function(_Ref), _Ref Function(_Ref)>(
      'SecTrustCopyCertificateChain',
    );
final _certificateData = _security
    .lookupFunction<_Ref Function(_Ref), _Ref Function(_Ref)>(
      'SecCertificateCopyData',
    );

final class _TrustFailure implements Exception {
  final int code;
  final String message;
  const _TrustFailure(this.code, this.message);
}

(Slice<GoString>, int, GoString) $verifySystemCertificates(
  Slice<GoString> certificates,
  GoString name,
  double date,
  bool hasDate,
) {
  if (!Platform.isMacOS && !Platform.isIOS) {
    return (
      Slice<GoString>.nil(),
      -1,
      GoString.ascii('platform certificate verification is unavailable'),
    );
  }
  final owned = <_Ref>[];
  _Ref own(_Ref ref, String operation) {
    if (ref == nullptr) {
      throw _TrustFailure(-1, '$operation returned null');
    }
    owned.add(ref);
    return ref;
  }

  _Ref data(GoString value, {bool string = false}) {
    final buffer = _malloc(value.length).cast<Uint8>();
    if (buffer == nullptr) {
      throw const _TrustFailure(-1, 'certificate buffer allocation failed');
    }
    try {
      buffer.asTypedList(value.length).setAll(0, value.bytes);
      return own(
        string
            ? _stringCreate(nullptr, buffer, value.length, 0x08000100, 0)
            : _dataCreate(nullptr, buffer, value.length),
        string ? 'CFStringCreateWithBytes' : 'CFDataCreate',
      );
    } finally {
      _free(buffer.cast());
    }
  }

  void status(int code, String operation) {
    if (code != 0) throw _TrustFailure(code, '$operation error: $code');
  }

  final result = _malloc(sizeOf<_Ref>()).cast<_Ref>();
  if (result == nullptr) {
    return (
      Slice<GoString>.nil(),
      -1,
      GoString.ascii('trust result allocation failed'),
    );
  }
  try {
    final certs = own(
      _arrayCreate(nullptr, 0, nullptr),
      'CFArrayCreateMutable',
    );
    for (var i = 0; i < certificates.len; i++) {
      final cert = own(
        _certificateCreate(nullptr, data(certificates[i])),
        'SecCertificateCreateWithData',
      );
      _arrayAppend(certs, cert);
    }
    final hostname = name.length == 0 ? nullptr : data(name, string: true);
    final policy = own(_policyCreate(1, hostname), 'SecPolicyCreateSSL');
    result.value = nullptr;
    status(
      _trustCreate(certs, policy, result),
      'SecTrustCreateWithCertificates',
    );
    final trust = own(result.value, 'SecTrustCreateWithCertificates');
    if (hasDate) {
      status(
        _trustDate(trust, own(_dateCreate(nullptr, date), 'CFDateCreate')),
        'SecTrustSetVerifyDate',
      );
    }
    result.value = nullptr;
    if (!_trustEvaluate(trust, result)) {
      final error = own(result.value, 'SecTrustEvaluateWithError');
      final description = own(
        _errorDescription(error),
        'CFErrorCopyDescription',
      );
      final bytes = own(
        _stringData(nullptr, description, 0x08000100, 0),
        'CFStringCreateExternalRepresentation',
      );
      final message = GoString.fromBytes(
        _dataBytes(bytes).asTypedList(_dataLength(bytes)),
      );
      return (Slice<GoString>.nil(), _errorCode(error), message);
    }
    final chain = own(_trustChain(trust), 'SecTrustCopyCertificateChain');
    final out = <GoString>[];
    for (var i = 0; i < _arrayCount(chain); i++) {
      final bytes = own(
        _certificateData(_arrayAt(chain, i)),
        'SecCertificateCopyData',
      );
      out.add(
        GoString.fromBytes(_dataBytes(bytes).asTypedList(_dataLength(bytes))),
      );
    }
    return (Slice<GoString>.of(out), 0, GoString.empty);
  } on _TrustFailure catch (error) {
    return (
      Slice<GoString>.nil(),
      error.code,
      GoString.fromString(error.message),
    );
  } finally {
    for (final ref in owned.reversed) {
      _release(ref);
    }
    _free(result.cast());
  }
}
