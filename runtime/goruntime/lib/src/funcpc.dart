import 'string.dart';

final _positions = <GoString>[];
final _counters = <GoString, int>{};
final _functions = Map<Function, int>.identity();

/// These counters identify source positions, not native instructions.
int $recordPC(GoString info) => _counters.putIfAbsent(info, () {
  _positions.add(info);
  return _positions.length;
});

GoString $pcInfo(int pc) =>
    pc > 0 && pc <= _positions.length ? _positions[pc - 1] : GoString.empty;

/// A strong map supports Wasm function references.
/// Register only named functions without captured values.
F $functionValue<F extends Function>(F value, GoString info) {
  _functions[value] ??= $recordPC(info);
  return value;
}

int? $functionPC(Function value) => _functions[value];
