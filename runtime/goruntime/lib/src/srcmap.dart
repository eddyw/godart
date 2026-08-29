/// The source map of the output.
///
/// The compiler writes 1 map for each package, and the init routine of the
/// package registers it. A key names a Dart file the way a stack trace names
/// it, and the value holds the functions and the lines of that file.
library;

import 'slice.dart';
import 'string.dart';

/// GoPos is a position of the Go source.
final class GoPos {
  final String function;
  final String file;
  final int line;

  const GoPos(this.function, this.file, this.line);

  @override
  String toString() => '$file:$line';
}

final Map<String, String> _data = <String, String>{};
final Map<String, _MappedFile> _parsed = <String, _MappedFile>{};

/// Registers the source map of 1 package. The list holds a key and the map of
/// that key, in pairs.
void $srcmap(List<String> data) {
  for (var i = 0; i + 1 < data.length; i += 2) {
    _data[data[i]] = data[i + 1];
  }
}

/// Gives the Go position of 1 line of the Dart output, and it gives null for a
/// line that the map does not hold.
GoPos? $goPos(String uri, int line) {
  final file = _lookup(uri);
  if (file == null) return null;
  return file.at(line);
}

_MappedFile? _lookup(String uri) {
  final hit = _parsed[uri];
  if (hit != null) return hit;
  var text = _data[uri];
  if (text == null) {
    for (final key in _data.keys) {
      if (uri.endsWith(key)) {
        text = _data[key];
        break;
      }
    }
  }
  if (text == null) return null;
  final out = _MappedFile.parse(text);
  _parsed[uri] = out;
  return out;
}

/// The functions and the lines of 1 Dart file. Both lists hold the lines in
/// ascending order, thus a read is a binary search.
final class _MappedFile {
  final List<int> funcAt;
  final List<String> funcFile;
  final List<String> funcName;
  final List<int> lineAt;
  final List<int> goLine;

  _MappedFile(
    this.funcAt,
    this.funcFile,
    this.funcName,
    this.lineAt,
    this.goLine,
  );

  factory _MappedFile.parse(String text) {
    final cut = text.indexOf('\n');
    final head = text.substring(0, cut);
    final body = text.substring(cut + 1);
    final funcAt = <int>[];
    final funcFile = <String>[];
    final funcName = <String>[];
    var at = 0;
    if (head.isNotEmpty) {
      for (final item in head.split(';')) {
        final first = item.indexOf('|');
        final second = item.indexOf('|', first + 1);
        at += int.parse(item.substring(0, first), radix: 36);
        funcAt.add(at);
        funcFile.add(item.substring(first + 1, second));
        funcName.add(item.substring(second + 1));
      }
    }
    final lineAt = <int>[];
    final goLine = <int>[];
    var dart = 0;
    var go = 0;
    if (body.isNotEmpty) {
      for (final item in body.split(',')) {
        var sign = item.indexOf('+');
        var step = 1;
        if (sign < 0) {
          sign = item.indexOf('-');
          step = -1;
        }
        dart += int.parse(item.substring(0, sign), radix: 36);
        go += step * int.parse(item.substring(sign + 1), radix: 36);
        lineAt.add(dart);
        goLine.add(go);
      }
    }
    return _MappedFile(funcAt, funcFile, funcName, lineAt, goLine);
  }

  GoPos? at(int line) {
    final i = _floor(lineAt, line);
    if (i < 0) return null;
    final f = _floor(funcAt, line);
    final name = f < 0 ? '' : funcName[f];
    final file = f < 0 ? '' : funcFile[f];
    return GoPos(name, file, goLine[i]);
  }
}

/// Gives the index of the last entry that is not above the value, and it gives
/// -1 when each entry is above it.
int _floor(List<int> list, int value) {
  var low = 0;
  var high = list.length - 1;
  var out = -1;
  while (low <= high) {
    final mid = (low + high) >> 1;
    if (list[mid] <= value) {
      out = mid;
      low = mid + 1;
    } else {
      high = mid - 1;
    }
  }
  return out;
}

/// Gives 1 item for each frame of the Go stack, in the form
/// "function|file|line". The count of skip drops the first frames, and frame 0
/// is the function that calls this one.
Slice<GoString> $stack(int skip) {
  final frames = $goStack(StackTrace.current);
  final out = <GoString>[];
  for (var i = skip; i < frames.length; i++) {
    final f = frames[i];
    out.add(GoString.fromString('${f.function}|${f.file}|${f.line}'));
  }
  return Slice.of(out);
}

/// Gives the Go position of each frame of a Dart stack trace. A frame that the
/// map does not hold stays out, thus the list holds the Go frames only.
List<GoPos> $goStack(StackTrace trace) {
  final out = <GoPos>[];
  for (final line in trace.toString().split('\n')) {
    final open = line.lastIndexOf('(');
    final close = line.lastIndexOf(')');
    if (open < 0 || close < open) continue;
    final inside = line.substring(open + 1, close);
    final col = inside.lastIndexOf(':');
    if (col < 0) continue;
    final row = inside.lastIndexOf(':', col - 1);
    if (row < 0) continue;
    final at = int.tryParse(inside.substring(row + 1, col));
    if (at == null) continue;
    final pos = $goPos(inside.substring(0, row), at);
    if (pos != null) out.add(pos);
  }
  return out;
}
