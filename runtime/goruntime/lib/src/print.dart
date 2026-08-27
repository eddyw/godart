import 'dart:io';

/// The Go builtin `print`. It writes to stderr with no separator.
void $print(List<String> parts) => stderr.write(parts.join());

/// The Go builtin `println`. It writes to stderr, and it puts 1 space between
/// 2 values.
void $println(List<String> parts) => stderr.writeln(parts.join(' '));

/// The text of a float, in the format that the Go builtin print uses.
///
/// Go writes the shortest decimal text that reads back as the same value. It
/// takes the exponent form when the exponent is below -4 or reaches 6.
String $fstring(double v) {
  if (v.isNaN) return 'NaN';
  if (v.isInfinite) return v > 0 ? '+Inf' : '-Inf';
  final String sign = v.isNegative ? '-' : '';
  if (v == 0) return '${sign}0';
  final String text = v.abs().toStringAsExponential();
  final int e = text.indexOf('e');
  String digits = text.substring(0, e).replaceAll('.', '');
  final int exponent = int.parse(text.substring(e + 1));
  while (digits.length > 1 && digits.endsWith('0')) {
    digits = digits.substring(0, digits.length - 1);
  }
  if (exponent < -4 || exponent >= 6) {
    final StringBuffer out = StringBuffer(sign)..write(digits[0]);
    if (digits.length > 1) out.write('.${digits.substring(1)}');
    out.write('e');
    out.write(exponent < 0 ? '-' : '+');
    out.write(exponent.abs().toString().padLeft(2, '0'));
    return out.toString();
  }
  if (exponent < 0) {
    return '${sign}0.${'0' * (-exponent - 1)}$digits';
  }
  if (digits.length > exponent + 1) {
    return '$sign${digits.substring(0, exponent + 1)}.${digits.substring(exponent + 1)}';
  }
  return '$sign$digits${'0' * (exponent + 1 - digits.length)}';
}
