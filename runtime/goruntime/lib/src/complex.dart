import 'int.dart';

/// A complex128 value. GoComplex32 retains the raw bits of complex64 parts.
base class GoComplex {
  final GoFloat64 real64;
  final GoFloat64 imag64;
  double get re => real64.toDouble();
  double get im => imag64.toDouble();

  GoComplex(double re, double im) : this.fromParts($f64(re), $f64(im));
  const GoComplex.fromParts(this.real64, this.imag64);

  static const GoComplex zero = GoComplex.fromParts(
    GoFloat64.zero,
    GoFloat64.zero,
  );

  GoComplex operator +(GoComplex o) => GoComplex(re + o.re, im + o.im);

  GoComplex operator -(GoComplex o) => GoComplex(re - o.re, im - o.im);

  GoComplex operator -() => GoComplex.fromParts(-real64, -imag64);

  GoComplex operator *(GoComplex o) =>
      GoComplex(re * o.re - im * o.im, re * o.im + im * o.re);

  /// The division of complex128div of the Go runtime: the algorithm of Smith,
  /// with the corrections of C99 for an infinity and for a zero.
  GoComplex operator /(GoComplex m) {
    double e, f;
    if (m.re.abs() >= m.im.abs()) {
      final double ratio = m.im / m.re;
      final double denom = m.re + ratio * m.im;
      e = (re + im * ratio) / denom;
      f = (im - re * ratio) / denom;
    } else {
      final double ratio = m.re / m.im;
      final double denom = m.im + ratio * m.re;
      e = (re * ratio + im) / denom;
      f = (im * ratio - re) / denom;
    }

    if (e.isNaN && f.isNaN) {
      double a = re, b = im;
      double c = m.re, d = m.im;
      if (c == 0 && d == 0 && (!a.isNaN || !b.isNaN)) {
        e = _copySign(double.infinity, c) * a;
        f = _copySign(double.infinity, c) * b;
      } else if ((a.isInfinite || b.isInfinite) && c.isFinite && d.isFinite) {
        a = _inf2one(a);
        b = _inf2one(b);
        e = double.infinity * (a * c + b * d);
        f = double.infinity * (b * c - a * d);
      } else if ((c.isInfinite || d.isInfinite) && a.isFinite && b.isFinite) {
        c = _inf2one(c);
        d = _inf2one(d);
        e = 0 * (a * c + b * d);
        f = 0 * (b * c - a * d);
      }
    }

    return GoComplex(e, f);
  }

  @override
  bool operator ==(Object other) =>
      other is GoComplex && re == other.re && im == other.im;

  @override
  int get hashCode => Object.hash(re, im);

  @override
  String toString() => '($re+${im}i)';
}

/// A signed 1 for an infinity, and a signed 0 for each other value.
double _inf2one(double f) => _copySign(f.isInfinite ? 1.0 : 0.0, f);

double _copySign(double magnitude, double sign) {
  final bool negative = sign.isNegative;
  return negative == magnitude.isNegative ? magnitude : -magnitude;
}

/// Rounds the 2 parts of a complex64 value.
GoComplex $c64(GoComplex c) =>
    c is GoComplex32 ? c : GoComplex32($f32(c.re), $f32(c.im));

GoComplex $c128(GoComplex c) => c is GoComplex32 ? GoComplex(c.re, c.im) : c;
GoComplex $complex32Zero() => GoComplex32.zero;

final class GoComplex32 extends GoComplex {
  final GoFloat32 real32;
  final GoFloat32 imag32;
  GoComplex32(this.real32, this.imag32)
    : super(real32.toDouble(), imag32.toDouble());
  static final zero = GoComplex32(GoFloat32.zero, GoFloat32.zero);
  @override
  GoComplex operator -() => GoComplex32(-real32, -imag32);
}
