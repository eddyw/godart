import 'string.dart';

final _positions = <GoString>[];
final _counters = <GoString, int>{};
final _functions = Expando<_FunctionPC>('function');

// Wasm Expando values need object references, not unboxed integer values.
final class _FunctionPC {
  final int pc;
  const _FunctionPC(this.pc);
}

final _entries = <String, int>{};
final _entryPCs = <int>{};

/// These counters identify source positions, not native instructions.
int $recordPC(GoString info) => _counters.putIfAbsent(info, () => _newPC(info));

int _newPC(GoString info) {
  _positions.add(info);
  // The adjacent odd value is the return PC for this source position.
  return _positions.length * 2;
}

GoString $pcInfo(int pc) {
  final index = (pc >> 1) - 1;
  return index >= 0 && index < _positions.length
      ? _positions[index]
      : GoString.empty;
}

/// Generic instances can share a displayed name but have distinct entries.
int $registerFunction(GoString info, [String? identity]) {
  final text = info.toDartString();
  return _entries.putIfAbsent(
    identity ?? text.substring(0, text.indexOf('|')),
    () {
      final pc = _newPC(info);
      _entryPCs.add(pc);
      return pc;
    },
  );
}

int $pcEntry(int pc) {
  pc &= ~1;
  if (_entryPCs.contains(pc)) return pc;
  final text = $pcInfo(pc).toDartString();
  if (text.isEmpty) return 0;
  return _entries[text.substring(0, text.indexOf('|'))] ?? pc;
}

F $functionValue<F extends Function>(F value, GoString info, [String? identity]) {
  _functions[value] ??= _FunctionPC($registerFunction(info, identity));
  return value;
}

int? $functionPC(Function value) => _functions[value]?.pc;

F $copyFunctionPC<F extends Function>(F value, Function? source) {
  _functions[value] = source == null ? null : _functions[source];
  return value;
}
