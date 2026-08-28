/// A box holds 1 value behind a reference.
///
/// The Dart form of a pointer to a struct is the object of the struct. Another
/// pointer needs a box, because a Dart int, double, bool, String, Slice, GoMap,
/// or Chan gives no reference to the variable that holds it.
///
/// The compiler writes a box for a variable whose address the code takes, thus
/// a program that takes no address holds no box.
final class Box<T> {
  T v;

  Box(this.v);
}
