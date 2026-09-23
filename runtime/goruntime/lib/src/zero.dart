import 'complex.dart';
import 'goint.dart';
import 'string.dart';

/// The zero value of each basic Go type, as a function with the type of the
/// value. The functions `$zeroInt` to `$zeroComplex` of iface.dart give an
/// `Object?`, thus a map of a typed element cannot take them.
///
/// A map, a channel, and an array each take a function that gives the zero
/// value of the element type. A tear off of 1 of these functions stands in
/// the place of a closure, thus the value costs no object of its own.
GoInt $intZero() => GoInt.zero;

bool $boolZero() => false;

GoComplex $complexZero() => GoComplex.zero;

GoString $stringZero() => GoString.empty;

/// The zero value of a pointer, a function, and an interface. The type Null
/// is a subtype of each nullable type, thus 1 function serves the 3.
Null $nullZero() => null;
