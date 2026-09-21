// This file includes code derived from the Go reflect package.
// Copyright 2009 The Go Authors. All rights reserved.
// Use of the Go-derived portions is governed by the BSD-style
// license in ../../LICENSE.

import 'goint.dart';
import 'iface.dart';

final Expando<(GoInt, int)> _layouts = Expando<(GoInt, int)>();

/// The logical Go layout uses 64-bit words on all Dart targets.
GoInt $typeSize(TypeDesc type) {
  switch (type.kind) {
    case 'bool' || 'int8' || 'uint8':
      return $gi(1);
    case 'int16' || 'uint16':
      return $gi(2);
    case 'int32' || 'uint32' || 'float32':
      return $gi(4);
    case 'int' ||
        'int64' ||
        'uint' ||
        'uint64' ||
        'uintptr' ||
        'float64' ||
        'complex64' ||
        'ptr' ||
        'unsafe.Pointer' ||
        'map' ||
        'chan' ||
        'func':
      return $gi(8);
    case 'complex128' || 'interface' || 'string':
      return $gi(16);
    case 'slice':
      return $gi(24);
    case 'array':
      return $gi(type.length) * $typeSize(type.elem!());
  }
  return _layout(type).$1;
}

int $typeAlign(TypeDesc type) {
  switch (type.kind) {
    case 'bool' || 'int8' || 'uint8':
      return 1;
    case 'int16' || 'uint16':
      return 2;
    case 'int32' || 'uint32' || 'float32' || 'complex64':
      return 4;
    case 'int' ||
        'int64' ||
        'uint' ||
        'uint64' ||
        'uintptr' ||
        'float64' ||
        'complex128' ||
        'ptr' ||
        'unsafe.Pointer' ||
        'map' ||
        'chan' ||
        'func' ||
        'interface' ||
        'string' ||
        'slice':
      return 8;
    case 'array':
      return $typeAlign(type.elem!());
  }
  return _layout(type).$2;
}

GoInt $fieldOffset(TypeDesc type, int index) {
  GoInt offset = GoInt.zero;
  for (int i = 0; i <= index; i++) {
    final TypeDesc field = type.fields[i].type();
    offset = _roundUp(offset, $typeAlign(field));
    if (i == index) return offset;
    offset += $typeSize(field);
  }
  return offset;
}

(GoInt, int) _layout(TypeDesc type) {
  final (GoInt, int)? cached = _layouts[type];
  if (cached != null) return cached;
  GoInt offset = GoInt.zero;
  GoInt last = GoInt.zero;
  int align = 1;
  for (final FieldDesc field in type.fields) {
    final TypeDesc desc = field.type();
    final int a = $typeAlign(desc);
    if (a > align) align = a;
    offset = _roundUp(offset, a);
    last = $typeSize(desc);
    offset += last;
  }
  // A trailing zero-sized field must not point beyond a nonempty struct.
  if (type.fields.isNotEmpty && last == GoInt.zero && offset > GoInt.zero) {
    offset += $gi(1);
  }
  final (GoInt, int) result = (_roundUp(offset, align), align);
  _layouts[type] = result;
  return result;
}

GoInt _roundUp(GoInt offset, int align) =>
    (offset + $gi(align - 1)) & -$gi(align);
