import 'iface.dart';

/// A weak reference and the description of the value that it names. The class
/// holds no strong link of the value.
final class _WeakRef {
  _WeakRef(this.ref, this.desc);

  final WeakReference<Object> ref;
  final TypeDesc desc;
}

/// The weak reference of each value, by value. 1 value gives 1 reference, thus
/// 2 weak pointers of Go that name 1 value compare equal.
///
/// An Expando drops the entry of a value that no other name reads, and the
/// reference inside the entry holds no strong link of that value.
final Expando<_WeakRef> _weakRefs = Expando<_WeakRef>('weak');

/// Gives the weak reference of the value inside v.
Object? $weakRef(Iface? v) {
  if (v == null) return null;
  final Object? target = v.value;
  if (target == null) return null;
  _WeakRef? held = _weakRefs[target];
  if (held == null) {
    held = _WeakRef(WeakReference<Object>(target), v.desc);
    _weakRefs[target] = held;
  }
  return held;
}

/// Gives the value of a weak reference, or null after a collection freed it.
Iface? $weakTarget(Object? r) {
  if (r is! _WeakRef) return null;
  final Object? target = r.ref.target;
  if (target == null) return null;
  return Iface(r.desc, target);
}
