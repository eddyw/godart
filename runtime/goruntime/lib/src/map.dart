import 'dart:collection';

import 'iface.dart';
import 'panic.dart';
import 'slice.dart';

/// An entry of a Go map. The entries form a list in the order of the writes,
/// and a range loop walks that list. A removed entry keeps its [next] link,
/// thus an iterator that stands on it still reaches the rest of the list.
final class _Entry<K, V> {
  final K key;
  V value;
  bool dead = false;
  _Entry<K, V>? prev;
  _Entry<K, V>? next;
  _Entry(this.key, this.value);
}

/// A Go map.
///
/// The map keeps the zero value of the element type, because a read of a key
/// that the map does not hold gives that value. The order of the keys follows
/// the order of the writes. Go gives no order, thus a Go program must not
/// depend on it.
final class GoMap<K, V> {
  final HashMap<K, _Entry<K, V>> _entries;
  final V Function() _zero;
  _Entry<K, V>? _head;
  _Entry<K, V>? _tail;

  /// True for the zero value of a map type. A read gives the zero value, and a
  /// write panics.
  final bool isNil;

  /// A key of an array type compares by value in Go, and a Dart list compares
  /// by identity, thus such a map takes its own equality and its own hash.
  GoMap(this._zero, {bool Function(K, K)? equals, int Function(K)? hashCode})
    : _entries = equals == null
          ? HashMap<K, _Entry<K, V>>()
          : HashMap<K, _Entry<K, V>>(equals: equals, hashCode: hashCode),
      isNil = false;

  GoMap.nil(this._zero) : _entries = HashMap<K, _Entry<K, V>>(), isNil = true;

  int get length => _entries.length;

  V get(K key) {
    final _Entry<K, V>? e = _entries[key];
    if (e == null) return _zero();
    return e.value;
  }

  /// The Go form `v, ok := m[k]`.
  (V, bool) get2(K key) {
    final _Entry<K, V>? e = _entries[key];
    if (e == null) return (_zero(), false);
    return (e.value, true);
  }

  void set(K key, V value) {
    if (isNil) {
      $panic(
        Iface(
          $typeGoError,
          const GoError('assignment to entry in nil map', plain: true),
        ),
      );
    }
    final _Entry<K, V>? e = _entries[key];
    if (e != null) {
      e.value = value;
      return;
    }
    final entry = _Entry(key, value);
    entry.prev = _tail;
    if (_tail == null) {
      _head = entry;
    } else {
      _tail!.next = entry;
    }
    _tail = entry;
    _entries[key] = entry;
  }

  void remove(K key) {
    final _Entry<K, V>? e = _entries.remove(key);
    if (e == null) return;
    _unlink(e);
  }

  void _unlink(_Entry<K, V> e) {
    e.dead = true;
    if (e.prev == null) {
      _head = e.next;
    } else {
      e.prev!.next = e.next;
    }
    if (e.next == null) {
      _tail = e.prev;
    } else {
      e.next!.prev = e.prev;
    }
    e.prev = null;
  }

  bool has(K key) => _entries.containsKey(key);

  void clear() {
    var e = _head;
    while (e != null) {
      e.dead = true;
      e.prev = null;
      e = e.next;
    }
    _head = null;
    _tail = null;
    _entries.clear();
  }

  /// A fresh hash for a NaN key. Each NaN entry stands alone, thus 1 shared
  /// hash would put each one into 1 bucket, and the map would grow in square
  /// time. The Go runtime also spreads NaN with random bits. A NaN key never
  /// reads back, thus the lookup needs no stable hash. Each step stays under
  /// 53 bits, thus the JS target computes the same walk.
  static int nanHash() {
    _nanSeed = (_nanSeed * 69069 + 12345) & 0xFFFFFF;
    return _nanSeed;
  }

  static int _nanSeed = 0x5F491;

  /// The keys, in the order of the writes. The format code, the json code,
  /// and the reflect code read this list. A range loop takes [GoMapIter].
  Slice<K> keys() {
    final list = <K>[];
    var e = _head;
    while (e != null) {
      list.add(e.key);
      e = e.next;
    }
    return Slice.of(list);
  }
}

/// The iterator of a range loop over a map. Go does not give a key that the
/// body removes, and the iterator jumps over each dead entry. A key that the
/// body adds joins the tail, and Go allows the loop to give it. The value
/// reads the entry at the visit, thus a NaN key needs no snapshot.
final class GoMapIter<K, V> {
  _Entry<K, V>? _at;
  _Entry<K, V>? _cur;

  GoMapIter(GoMap<K, V> m) : _at = m._head;

  /// The first value of a frame local. The machine writes the true iterator
  /// before the loop starts.
  GoMapIter.empty() : _at = null;

  bool next() {
    var e = _at;
    while (e != null && e.dead) {
      e = e.next;
    }
    if (e == null) {
      _at = null;
      return false;
    }
    _cur = e;
    _at = e.next;
    return true;
  }

  K get key => _cur!.key;
  V get value => _cur!.value;
}
