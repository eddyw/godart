import 'dart:ffi';

final _library = DynamicLibrary.process();
// Flutter engines can hide the embedder API.
final _collect = _library.providesSymbol('Dart_NotifyDestroyed')
    ? _library.lookupFunction<Void Function(), void Function()>(
        'Dart_NotifyDestroyed',
      )
    : null;

bool $collectGarbage() {
  final collect = _collect;
  if (collect == null) return false;
  // This call requires a current isolate and a non-leaf FFI transition.
  collect();
  return true;
}
