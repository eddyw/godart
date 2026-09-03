import 'panic.dart';
import 'platform_native.dart' if (dart.library.js_interop) 'platform_web.dart';
import 'srcmap.dart';
import 'string.dart';

/// The bytes that a capture of standard output holds. Null when no capture
/// runs.
List<int>? _captured;

/// Starts a capture of standard output, or ends it and gives the text.
///
/// The harness of an example reads it: Go holds the output of the example and
/// compares it with the text of the comment.
GoString $captureOut(bool on) {
  if (on) {
    _captured = <int>[];
    return GoString.empty;
  }
  final List<int>? held = _captured;
  _captured = null;
  if (held == null) return GoString.empty;
  return GoString.fromBytes(held);
}

/// Writes text to standard output. The package fmt uses it.
///
/// The bytes reach the sink with no step through a Dart String, thus a byte
/// that holds no rune of UTF-8 keeps its value.
void $stdout(GoString text) {
  final List<int>? held = _captured;
  if (held != null) {
    held.addAll(text.bytes);
    return;
  }
  $platformStdout(text.bytes);
}

/// The Go builtin `print`. It writes to stderr with no separator.
void $print(List<String> parts) => $platformStderrText(parts.join());

/// The Go builtin `println`. It writes to stderr, and it puts 1 space between
/// 2 values.
void $println(List<String> parts) =>
    $platformStderrText('${parts.join(' ')}\n');

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

/// Reports a panic that no function stops, and ends the program. Go writes the
/// value to stderr and exits with the status 2.
Never $fatal(GoPanic panic, StackTrace trace) {
  $platformStderrText('panic: ${panic.value}\n\ngoroutine 1 [running]:\n');
  for (final GoPos pos in $goStack(trace)) {
    $platformStderrText('${pos.function}()\n\t${pos.file}:${pos.line}\n');
  }
  $platformExit(2);
}
