import 'dart:convert';
import 'dart:io';

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
  Activity(this.name, this.description, this.locationName, this.duration,
      this.timeOfDay, this.familyFriendly, this.price, this.destinationRef,
      this.ref, this.imageUrl);

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
    final String s =
        jsonEncode(typed.map((Activity a) => a.toJson()).toList());
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

  // The same cases, in the Dart that gart wrote.
  measure('bytes.count', 'gart', () => go.CountBytes().toInt());
  measure('json.scan', 'gart', () => go.Scan().toInt());
  measure('json.decode.any', 'gart', () => go.DecodeAny().toInt());
  measure('json.decode.typed', 'gart', () => go.DecodeTyped().toInt());
  measure('json.encode.typed', 'gart', () => go.EncodeTyped().toInt());
  measure('str.concat.5k', 'gart', () => go.Concat(GoInt(5000)).toInt());
  measure('str.builder.200k', 'gart', () => go.Builder(GoInt(200000)).toInt());
  measure('str.substr.100k', 'gart', () => go.Substr(GoInt(100000)).toInt());

  // The boundary. A Go string holds UTF-8 bytes, and a Dart string holds UTF-16
  // code units, thus each crossing encodes or decodes the whole text.
  for (final int len in <int>[20, 1000, 100000]) {
    final GoString gs = go.Text(GoInt(len));
    final String ds = gs.toDartString();
    measure('cross.todart.$len', 'gart', () => gs.toDartString().length);
    measure('cross.togo.$len', 'gart',
        () => go.Take(GoString.fromString(ds)).toInt());
  }
}
