import 'dart:collection';

import 'panic.dart';
import 'slice.dart';

/// A Go map.
///
/// The map keeps the zero value of the element type, because a read of a key
/// that the map does not hold gives that value. The order of the keys follows
/// the order of the writes. Go gives no order, thus a Go program must not
/// depend on it.
final class GoMap<K, V> {
  final LinkedHashMap<K, V> _entries = LinkedHashMap<K, V>();
  final V Function() _zero;

  /// True for the zero value of a map type. A read gives the zero value, and a
  /// write panics.
  final bool isNil;

  GoMap(this._zero) : isNil = false;

  GoMap.nil(this._zero) : isNil = true;

  int get length => _entries.length;

  V get(K key) {
    final V? value = _entries[key];
    if (value == null && !_entries.containsKey(key)) return _zero();
    return value as V;
  }

  /// The Go form `v, ok := m[k]`.
  (V, bool) get2(K key) {
    final V? value = _entries[key];
    if (value == null && !_entries.containsKey(key)) return (_zero(), false);
    return (value as V, true);
  }

  void set(K key, V value) {
    if (isNil) $panic(const GoError('assignment to entry in nil map'));
    _entries[key] = value;
  }

  void remove(K key) => _entries.remove(key);

  bool has(K key) => _entries.containsKey(key);

  void clear() => _entries.clear();

  /// The keys at the start of a range loop. Go does not give a key that the
  /// body removes, thus the loop reads the map again for each key.
  Slice<K> keys() => Slice.of(_entries.keys.toList());
}
