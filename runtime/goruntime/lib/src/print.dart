import 'format.dart';
import 'goint.dart';
import 'iface.dart';
import 'panic.dart';
import 'platform_native.dart' if (dart.library.js_interop) 'platform_web.dart';
import 'reflect.dart';
import 'slice.dart';
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

String $referenceText(Iface? value) {
  final String pointer = '0x${$valuePointer(value).toRadixString(16)}';
  final Object? data = value?.value;
  return data is Slice<Object?> ? '[${data.len}/${data.cap}]$pointer' : pointer;
}

String $interfaceText(Iface? value) {
  if (value == null) return '(0x0,0x0)';
  return '(0x${identityHashCode(value.desc).toRadixString(16)},'
      '0x${$valuePointer(value).toRadixString(16)})';
}

String $genericPrintText(Object? value, TypeDesc desc) {
  switch (desc.kind) {
    case 'string':
      return (value as GoString).toDartString();
    case 'float32':
    case 'float64':
    case 'complex64':
    case 'complex128':
      return $goFormat(Iface(desc, value), GoString.ascii('g')).toDartString();
    case 'uint':
    case 'uint64':
    case 'uintptr':
      return $ustring(value as GoInt);
    case 'bool':
    case 'int':
    case 'int8':
    case 'int16':
    case 'int32':
    case 'int64':
    case 'uint8':
    case 'uint16':
    case 'uint32':
      return value.toString();
    case 'interface':
      return $interfaceText(value as Iface?);
    default:
      return $referenceText(Iface(desc, value));
  }
}

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
  $platformStderrText(
    '${$panicChain(panic)}\ngoroutine ${panic.goroutineID} [running]:\n',
  );
  for (final GoPos pos in $panicStack(panic)) {
    $platformStderrText(
      '${pos.function == 'runtime.gopanic' ? 'panic' : pos.function}()\n\t${pos.file}:${pos.line}\n',
    );
  }
  $platformExit(2);
}
