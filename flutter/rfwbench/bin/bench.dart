// The measure of the per frame cost of A9, package:rfw.
//
// A9 writes the user interface as a data blob. The Go side builds the blob,
// and the Dart host decodes it and renders it with the rfw runtime. A frame
// that changes the tree thus pays an encode and a decode.
//
// The frame budget at 60 hertz is 16,700 microseconds. This program gives the
// part of that budget that the pair takes, over a sweep of the node count.
//
// The program reads no Flutter library. package:rfw/formats.dart holds the
// wire format alone, thus `dart run` measures it with no engine.
import 'dart:typed_data';

import 'package:rfw/formats.dart';

/// The text of a library of n rows. Each row holds a Text widget inside a
/// Padding, which is the shape of a list of a real application.
String libraryText(int n) {
  final StringBuffer b = StringBuffer('import core;\nimport material;\n');
  b.write('widget root = Column(children: [\n');
  for (var i = 0; i < n; i++) {
    b.write('  Padding(padding: [8.0, 4.0, 8.0, 4.0], '
        'child: Text(text: ["row ", "$i"])),\n');
  }
  b.write(']);\n');
  return b.toString();
}

/// The middle value of a list of measurements.
double median(List<double> xs) {
  xs.sort();
  final int i = xs.length ~/ 2;
  if (xs.length.isOdd) return xs[i];
  return (xs[i - 1] + xs[i]) / 2;
}

/// Runs body until 200 milliseconds pass, and gives the microseconds of 1 run.
double time(void Function() body) {
  body();
  final Stopwatch w = Stopwatch()..start();
  var n = 0;
  while (w.elapsedMilliseconds < 200) {
    body();
    n++;
  }
  w.stop();
  return w.elapsedMicroseconds / n;
}

void main() {
  const List<int> counts = <int>[10, 50, 100, 200, 500, 1000];
  const int runs = 5;
  print('nodes,bytes,encode_us,decode_us,total_us,budget_pct');
  for (final int n in counts) {
    final RemoteWidgetLibrary lib = parseLibraryFile(libraryText(n));
    final Uint8List blob = encodeLibraryBlob(lib);
    final List<double> enc = <double>[];
    final List<double> dec = <double>[];
    for (var r = 0; r < runs; r++) {
      enc.add(time(() => encodeLibraryBlob(lib)));
      dec.add(time(() => decodeLibraryBlob(blob)));
    }
    final double e = median(enc);
    final double d = median(dec);
    final double total = e + d;
    print('$n,${blob.length},${e.toStringAsFixed(1)},'
        '${d.toStringAsFixed(1)},${total.toStringAsFixed(1)},'
        '${(total / 16700 * 100).toStringAsFixed(2)}');
  }
}
