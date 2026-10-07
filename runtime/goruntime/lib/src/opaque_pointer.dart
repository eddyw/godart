// This file includes code derived from the Go reflect package.
// Copyright 2009 The Go Authors. All rights reserved.
// Use of the Go-derived portions is governed by the BSD-style
// license in ../../LICENSE.

import 'array.dart';
import 'box.dart';
import 'goint.dart';
import 'iface.dart';
import 'layout.dart';
import 'slice.dart';

abstract interface class $GoStruct {}

final Expando<(Object, int)> _inlineLocations = Expando('inline locations');
final Expando<GoInt> _allocationSizes = Expando('logical allocation sizes');
final Expando<GoInt> _allocationNumbers = Expando('logical allocation numbers');
// The high bit separates data identities from nonnegative synthetic source PCs.
GoInt _nextAllocationNumber = $gi2(-2147483648, 0);

void $bindStruct(Object value, List<TypeDesc> types, List<Object?> fields) {
  var offset = 0;
  var alignment = 1;
  var lastSize = 0;
  for (var index = 0; index < types.length; index++) {
    final type = types[index];
    final align = $typeAlign(type);
    if (align > alignment) alignment = align;
    offset = (offset + align - 1) & -align;
    if (type.kind == 'struct' || type.kind == 'array') {
      _inlineLocations[$pointerOwner(fields[index])!] = (value, offset);
    }
    lastSize = $ix($typeSize(type));
    offset += lastSize;
  }
  if (types.isNotEmpty && lastSize == 0 && offset != 0) offset++;
  _allocationSizes[value] = $gi((offset + alignment - 1) & -alignment);
}

List<T> $bindArray<T>(List<T> values, TypeDesc element) {
  _allocationSizes[values] = $arrayLength(values) * $typeSize(element);
  if (element.kind != 'struct' && element.kind != 'array') return values;
  if (values is $ZeroList<T>) {
    if (values.$length != GoInt.zero) {
      _inlineLocations[$pointerOwner(values.value)!] = (values, 0);
    }
    return values;
  }
  final size = $ix($typeSize(element));
  for (var index = 0; index < values.length; index++) {
    _inlineLocations[$pointerOwner(values[index])!] = (values, index * size);
  }
  return values;
}

void $bindBox(Object owner, Object? value) {
  if (value is $GoStruct || value is List) {
    _inlineLocations[$pointerOwner(value)!] = (owner, 0);
    final size = _allocationSizes[$pointerOwner(value)!];
    if (size != null) _allocationSizes[owner] = size;
  }
}

final class $OpaquePointer {
  final Object allocation;
  final int offset;
  $OpaquePointer(this.allocation, this.offset);

  @override
  bool operator ==(Object other) =>
      other is $OpaquePointer &&
      identical(allocation, other.allocation) &&
      offset == other.offset;

  @override
  int get hashCode => Object.hash(identityHashCode(allocation), offset);
}

// Inline locations must exist before a pointer becomes a map key.
$OpaquePointer? $opaquePointer(Object? pointer, TypeDesc element) {
  if (pointer == null) return null;
  var offset = 0;
  Object owner;
  if (pointer is $Ref) {
    owner = $pointerOwner(pointer.owner)!;
    final struct = pointer.structType;
    if (struct != null) {
      offset = $ix($fieldOffset(struct, pointer.index));
      _allocationSizes[owner] ??= $typeSize(struct);
    } else if (owner is List && owner is! $StructView) {
      offset = pointer.index * $ix($typeSize(element));
    }
  } else {
    owner = $pointerOwner(pointer)!;
  }
  if (owner is List && owner is! $ArrayView) {
    final item = element.kind == 'array' && pointer is! $Ref
        ? element.elem!()
        : element;
    _allocationSizes[owner] ??= $arrayLength(owner) * $typeSize(item);
  }
  if (owner is $ArrayView) {
    offset += owner.off * $ix($typeSize(element.elem!()));
    owner = owner.array;
    _allocationSizes[owner] ??=
        $arrayLength(owner) * $typeSize(element.elem!());
  }
  return $opaqueStoragePointer(owner, offset, $typeSize(element));
}

$OpaquePointer $opaqueStoragePointer(Object owner, int offset, GoInt size) {
  _allocationSizes[owner] ??= size;
  while (owner is! String) {
    final location = _inlineLocations[owner];
    if (location == null) break;
    owner = location.$1;
    offset += location.$2;
  }
  return $OpaquePointer(owner, offset);
}

final Map<int, Object> _codeOwners = {};

// Function pointers retain the existing synthetic source PC contract.
$OpaquePointer $opaqueCodePointer(int pc) {
  final owner = _codeOwners.putIfAbsent(pc, Object.new);
  _allocationNumbers[owner] = $gi(pc);
  return $OpaquePointer(owner, 0);
}

// Reserve the actual logical span. Numeric identities preserve inline offset order.
// These numbers name managed allocations and expose no machine address.
GoInt $opaquePointerNumber($OpaquePointer pointer) {
  final owner = pointer.allocation;
  var number = _allocationNumbers[owner];
  if (number == null) {
    // The logical Go layout requires at most 8-byte allocation alignment.
    number = (_nextAllocationNumber + $gi(7)) & -$gi(8);
    _allocationNumbers[owner] = number;
    _nextAllocationNumber =
        number + (_allocationSizes[owner] ?? GoInt.zero) + $gi(1);
  }
  return number + $gi(pointer.offset);
}
