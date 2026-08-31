import 'dart:collection';

import 'iface.dart';
import 'panic.dart';
import 'slice.dart';

/// A Go map.
///
/// The map keeps the zero value of the element type, because a read of a key
/// that the map does not hold gives that value. The order of the keys follows
/// the order of the writes. Go gives no order, thus a Go program must not
/// depend on it.
final class GoMap<K, V> {
  final LinkedHashMap<K, V> _entries;
  final V Function() _zero;

  /// True for the zero value of a map type. A read gives the zero value, and a
  /// write panics.
  final bool isNil;

  /// A key of an array type compares by value in Go, and a Dart list compares
  /// by identity, thus such a map takes its own equality and its own hash.
  GoMap(this._zero, {bool Function(K, K)? equals, int Function(K)? hashCode})
    : _entries = equals == null
          ? LinkedHashMap<K, V>()
          : LinkedHashMap<K, V>(equals: equals, hashCode: hashCode),
      isNil = false;

  GoMap.nil(this._zero) : _entries = LinkedHashMap<K, V>(), isNil = true;

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
    if (isNil) {
      $panic(
        Iface($typeGoError, const GoError('assignment to entry in nil map')),
      );
    }
    _entries[key] = value;
  }

  void remove(K key) => _entries.remove(key);

  bool has(K key) => _entries.containsKey(key);

  void clear() {
    _entries.clear();
    clearCount++;
  }

  /// The count of the clear calls. A NaN key never reads back, and only a
  /// clear removes its entry, thus a range loop compares this count to see
  /// whether such an entry still stands.
  int clearCount = 0;

  /// The values at the start of a range loop, in the order of [keys]. A NaN
  /// key never reads back, thus the loop takes its value from this snapshot.
  Slice<V> values() => Slice.of(_entries.values.toList());

  /// A fresh hash for a NaN key. Each NaN entry stands alone, thus 1 shared
  /// hash would put each one into 1 bucket, and the map would grow in square
  /// time. The Go runtime also spreads NaN with random bits. A NaN key never
  /// reads back, thus the lookup needs no stable hash.
  static int nanHash() {
    _nanSeed = _nanSeed * 6364136223846793005 + 1442695040888963407;
    return _nanSeed >>> 33;
  }

  static int _nanSeed = 0x9e3779b97f4a7c15;

  /// The keys at the start of a range loop. Go does not give a key that the
  /// body removes, thus the loop reads the map again for each key.
  Slice<K> keys() => Slice.of(_entries.keys.toList());
}
