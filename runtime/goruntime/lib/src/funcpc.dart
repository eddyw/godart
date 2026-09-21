import 'string.dart';

final _positions = <GoString>[];
final _counters = <GoString, int>{};
final _functions = Map<Function, int>.identity();
final _entries = <String, int>{};

/// These counters identify source positions, not native instructions.
int $recordPC(GoString info) => _counters.putIfAbsent(info, () => _newPC(info));

int _newPC(GoString info) {
  _positions.add(info);
  return _positions.length;
}

GoString $pcInfo(int pc) =>
    pc > 0 && pc <= _positions.length ? _positions[pc - 1] : GoString.empty;

int $registerFunction(GoString info) {
  final text = info.toDartString();
  return _entries.putIfAbsent(
    text.substring(0, text.indexOf('|')),
    () => _newPC(info),
  );
}

int $pcEntry(int pc) {
  final text = $pcInfo(pc).toDartString();
  if (text.isEmpty) return 0;
  return _entries[text.substring(0, text.indexOf('|'))] ?? pc;
}

/// A strong map supports Wasm function references.
/// Register only named functions without captured values.
F $functionValue<F extends Function>(F value, GoString info) {
  _functions[value] ??= $registerFunction(info);
  return value;
}

int? $functionPC(Function value) => _functions[value];
