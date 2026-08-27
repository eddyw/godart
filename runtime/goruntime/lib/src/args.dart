import 'slice.dart';
import 'string.dart';

/// The arguments of the program. The entry point writes them.
List<String> $argv = const <String>[];

/// Holds the arguments of the program. The generated entry point calls it.
void $setArgs(List<String> arguments) {
  $argv = List<String>.unmodifiable(arguments);
}

/// The arguments of the program, without the name of the program.
Slice<GoString> $args() =>
    Slice<GoString>.of($argv.map(GoString.fromString).toList());
