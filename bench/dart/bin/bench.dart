import 'dart:convert';
import 'dart:io';
import 'dart:math' as math;
import 'dart:typed_data';

import 'package:cases/cases.dart' as go;
import 'package:goruntime/goruntime.dart';

/// This program runs each case 2 times: 1 time in Dart, and 1 time in the Dart
/// that gart wrote from the Go source of bench/go. It also measures the 2
/// crossings of the boundary of a string.
///
/// It writes 1 CSV line for each case: the name, the runtime, the nanoseconds
/// of 1 operation, and the count of the operations.
///
/// Run it as an ahead of time build. The warm up of the Dart VM hides the
/// result of a short case.

const int budgetMs = 300;
const String piece = '0123456789';

const List<String> numbers = <String>[
  '0', '7', '42', '1234', '99999', '2147483647', '-58', '300', //
];
const List<String> decimals = <String>[
  '0.5', '3.14159', '-2.5', '1e3', '12345.6789', '0.0001', //
];

/// Point holds the 4 numbers of a rectangle, exactly as the Go struct does.
class Point {
  Point(this.x, this.y, this.w, this.h);

  final double x;
  final double y;
  final double w;
  final double h;
}

void measure(String name, String runtime, int Function() f) {
  f();
  final Stopwatch sw = Stopwatch()..start();
  int n = 0;
  while (sw.elapsedMilliseconds < budgetMs) {
    f();
    n++;
  }
  sw.stop();
  final int each = (sw.elapsedMicroseconds * 1000) ~/ n;
  print('$name,$runtime,$each,$n');
}

class Activity {
  Activity(
    this.name,
    this.description,
    this.locationName,
    this.duration,
    this.timeOfDay,
    this.familyFriendly,
    this.price,
    this.destinationRef,
    this.ref,
    this.imageUrl,
  );

  factory Activity.fromJson(Map<String, dynamic> m) => Activity(
    m['name'] as String,
    m['description'] as String,
    m['locationName'] as String,
    (m['duration'] as num).toDouble(),
    m['timeOfDay'] as String,
    m['familyFriendly'] as bool,
    (m['price'] as num).toDouble(),
    m['destinationRef'] as String,
    m['ref'] as String,
    m['imageUrl'] as String,
  );

  final String name;
  final String description;
  final String locationName;
  final double duration;
  final String timeOfDay;
  final bool familyFriendly;
  final double price;
  final String destinationRef;
  final String ref;
  final String imageUrl;

  Map<String, dynamic> toJson() => <String, dynamic>{
    'name': name,
    'description': description,
    'locationName': locationName,
    'duration': duration,
    'timeOfDay': timeOfDay,
    'familyFriendly': familyFriendly,
    'price': price,
    'destinationRef': destinationRef,
    'ref': ref,
    'imageUrl': imageUrl,
  };
}

void main(List<String> args) {
  if (args.isEmpty) {
    stderr.writeln('usage: bench <path of activities.json>');
    exit(2);
  }
  final String path = args[0];
  final List<int> bytes = File(path).readAsBytesSync();
  final String text = utf8.decode(bytes);
  List<Activity> typed = <Activity>[];

  go.$init();
  go.Load(GoString.fromString(path));
  go.DecodeTyped();

  // The Dart cases. jsonDecode takes a string that stands already, thus the
  // fused case gives the number of an application that reads a file or a
  // socket.
  measure('bytes.count', 'dart', () {
    int n = 0;
    for (int i = 0; i < bytes.length; i++) {
      if (bytes[i] == 44) n++;
    }
    return n;
  });
  measure('json.decode.any', 'dart', () {
    final List<dynamic> v = jsonDecode(text) as List<dynamic>;
    return v.length;
  });
  measure('json.decode.any.bytes', 'dart', () {
    final List<dynamic> v =
        utf8.decoder.fuse(json.decoder).convert(bytes) as List<dynamic>;
    return v.length;
  });
  measure('json.decode.typed', 'dart', () {
    final List<dynamic> v = jsonDecode(text) as List<dynamic>;
    typed = v
        .map((dynamic e) => Activity.fromJson(e as Map<String, dynamic>))
        .toList();
    return typed.length;
  });
  measure('json.encode.typed', 'dart', () {
    final String s = jsonEncode(typed.map((Activity a) => a.toJson()).toList());
    return s.length;
  });
  measure('str.concat.5k', 'dart', () {
    String s = '';
    for (int i = 0; i < 5000; i++) {
      s += piece;
    }
    return s.length;
  });
  measure('str.builder.200k', 'dart', () {
    final StringBuffer b = StringBuffer();
    for (int i = 0; i < 200000; i++) {
      b.write(piece);
    }
    return b.length;
  });
  measure('str.substr.100k', 'dart', () {
    final StringBuffer b = StringBuffer();
    final int span = text.length - 21;
    for (int i = 0; i < 100000; i++) {
      final int off = (i * 977) % span;
      b.write(text.substring(off, off + 20));
    }
    return b.length;
  });

  // The scalar cases in Dart. A Dart int holds 64 bits, exactly as a Go int
  // does, thus intHash wraps at the same place in the 2 runtimes.
  measure('int.sum.1m', 'dart', () {
    int s = 0;
    for (int i = 0; i < 1000000; i++) {
      s += i;
    }
    return s;
  });
  measure('int.hash.1m', 'dart', () {
    int h = 0x1000193;
    for (int i = 0; i < 1000000; i++) {
      h = (h ^ i) * 0x100000001B3;
    }
    return h;
  });
  measure('int.div.1m', 'dart', () {
    int s = 0;
    for (int i = 1; i <= 1000000; i++) {
      s += 1000003 ~/ i + 1000003 % i;
    }
    return s;
  });
  measure('int.parse.100k', 'dart', () {
    int s = 0;
    for (int i = 0; i < 100000; i++) {
      s += int.parse(numbers[i % numbers.length]);
    }
    return s;
  });
  measure('int.format.100k', 'dart', () {
    int s = 0;
    for (int i = 0; i < 100000; i++) {
      s += (i * 7919).toString().length;
    }
    return s;
  });
  measure('float.sum.1m', 'dart', () {
    double s = 0;
    for (int i = 0; i < 1000000; i++) {
      s += i * 0.5;
    }
    return s.toInt();
  });
  measure('float.math.1m', 'dart', () {
    double s = 0;
    for (int i = 0; i < 1000000; i++) {
      final double x = i * 0.001;
      s += math.sqrt(x) * 1.5 + x * x;
    }
    return s.toInt();
  });
  measure('float.parse.100k', 'dart', () {
    int s = 0;
    for (int i = 0; i < 100000; i++) {
      s += double.parse(decimals[i % decimals.length]).toInt();
    }
    return s;
  });
  measure('float.format.100k', 'dart', () {
    int s = 0;
    for (int i = 0; i < 100000; i++) {
      s += (i * 1.5).toStringAsFixed(3).length;
    }
    return s;
  });
  measure('slice.append.1m', 'dart', () {
    final List<int> s = <int>[];
    for (int i = 0; i < 1000000; i++) {
      s.add(i);
    }
    return s.length;
  });
  measure('slice.sort.100k', 'dart', () {
    final List<int> s = List<int>.filled(100000, 0);
    int x = 1;
    for (int i = 0; i < 100000; i++) {
      x = (x * 1103515245 + 12345) & 0x7FFFFFFF;
      s[i] = x;
    }
    s.sort();
    return s[50000];
  });
  measure('map.string.100k', 'dart', () {
    final Map<String, int> m = <String, int>{};
    for (int i = 0; i < 100000; i++) {
      m[i.toString()] = i;
    }
    int s = 0;
    for (int i = 0; i < 100000; i++) {
      s += m[i.toString()]!;
    }
    return s;
  });
  measure('map.int.100k', 'dart', () {
    final Map<int, int> m = <int, int>{};
    for (int i = 0; i < 100000; i++) {
      m[i * 7919] = i;
    }
    int s = 0;
    for (int i = 0; i < 100000; i++) {
      s += m[i * 7919]!;
    }
    return s;
  });
  measure('struct.alloc.1m', 'dart', () {
    double s = 0;
    for (int i = 0; i < 1000000; i++) {
      final Point p = Point(i.toDouble(), i * 2.0, 10, 20);
      s += p.w * p.h + p.x + p.y;
    }
    return s.toInt();
  });

  // The same cases, in the Dart that gart wrote.
  measure('bytes.count', 'gart', () => go.CountBytes().toInt());
  measure('json.scan', 'gart', () => go.Scan().toInt());
  measure('json.decode.any', 'gart', () => go.DecodeAny().toInt());
  measure('json.decode.typed', 'gart', () => go.DecodeTyped().toInt());
  measure('json.encode.typed', 'gart', () => go.EncodeTyped().toInt());
  measure('str.concat.5k', 'gart', () => go.Concat(GoInt(5000)).toInt());
  measure('str.builder.200k', 'gart', () => go.Builder(GoInt(200000)).toInt());
  measure('str.substr.100k', 'gart', () => go.Substr(GoInt(100000)).toInt());

  measure('int.sum.1m', 'gart', () => go.IntSum(GoInt(1000000)).toInt());
  measure('int.hash.1m', 'gart', () => go.IntHash(GoInt(1000000)).toInt());
  measure('int.div.1m', 'gart', () => go.IntDiv(GoInt(1000000)).toInt());
  measure('int.parse.100k', 'gart', () => go.IntParse(GoInt(100000)).toInt());
  measure('int.format.100k', 'gart', () => go.IntFormat(GoInt(100000)).toInt());
  measure('float.sum.1m', 'gart', () => go.FloatSum(GoInt(1000000)).toInt());
  measure('float.math.1m', 'gart', () => go.FloatMath(GoInt(1000000)).toInt());
  measure(
    'float.parse.100k',
    'gart',
    () => go.FloatParse(GoInt(100000)).toInt(),
  );
  measure(
    'float.format.100k',
    'gart',
    () => go.FloatFormat(GoInt(100000)).toInt(),
  );
  measure(
    'slice.append.1m',
    'gart',
    () => go.SliceAppend(GoInt(1000000)).toInt(),
  );
  measure('slice.sort.100k', 'gart', () => go.SliceSort(GoInt(100000)).toInt());
  measure('map.string.100k', 'gart', () => go.MapString(GoInt(100000)).toInt());
  measure('map.int.100k', 'gart', () => go.MapInt(GoInt(100000)).toInt());
  measure(
    'struct.alloc.1m',
    'gart',
    () => go.StructAlloc(GoInt(1000000)).toInt(),
  );

  // The boundary. A Go string holds UTF-8 bytes, and a Dart string holds UTF-16
  // code units, thus each crossing encodes or decodes the whole text.
  //
  // Each name says whether the read meets 1 GoString again:
  //
  // - `cross.todart` reads 1 object again at each operation. A memo of the
  //   decode would hold after the first read.
  // - `cross.fresh` builds a new object at each operation. A memo would never
  //   hold, and the case gives the cost of the extra field.
  // - `cross.frame` reads 500 objects 1 time each, exactly as 1 rebuild of a
  //   screen does. A memo would hold from the second operation.
  for (final int len in <int>[20, 1000, 100000, 2000000]) {
    final GoString gs = go.Text(GoInt(len));
    final String ds = gs.toDartString();
    final Uint8List raw = Uint8List.fromList(gs.bytes);
    measure('cross.todart.$len', 'gart', () => gs.toDartString().length);
    measure(
      'cross.fresh.$len',
      'gart',
      () => GoString.take(Uint8List.view(raw.buffer)).toDartString().length,
    );
    measure(
      'cross.togo.$len',
      'gart',
      () => go.Take(GoString.fromString(ds)).toInt(),
    );
  }
  for (final (int count, int len) in <(int, int)>[(500, 20), (200, 500)]) {
    final Slice<GoString> labels = go.Labels(GoInt(count), GoInt(len));
    measure('cross.frame.${count}x$len', 'gart', () {
      int n = 0;
      for (int i = 0; i < labels.len; i++) {
        n += labels[i].toDartString().length;
      }
      return n;
    });
  }
}
