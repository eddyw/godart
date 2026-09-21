import 'goint.dart';
import 'string.dart';

final _counts = <String, GoInt>{};
bool $godebugRegistered = false;

void $registerGODEBUG() {
  $godebugRegistered = true;
}

void $incGODEBUG(GoString name) {
  final key = name.toDartString();
  _counts[key] = (_counts[key] ?? $gi(0)) + $gi(1);
}

GoInt $godebugCount(GoString name) => _counts[name.toDartString()] ?? $gi(0);
