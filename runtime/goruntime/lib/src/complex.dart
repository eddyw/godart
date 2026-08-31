import 'int.dart';

/// A Go complex number. The class holds the 2 parts of complex128, and a
/// complex64 value rounds each part with [$f32] after each operation.
final class GoComplex {
  final double re;
  final double im;

  const GoComplex(this.re, this.im);

  static const GoComplex zero = GoComplex(0, 0);

  GoComplex operator +(GoComplex o) => GoComplex(re + o.re, im + o.im);

  GoComplex operator -(GoComplex o) => GoComplex(re - o.re, im - o.im);

  GoComplex operator -() => GoComplex(-re, -im);

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
GoComplex $c64(GoComplex c) => GoComplex($f32(c.re), $f32(c.im));
