import 'package:goruntime/goruntime.dart';
import 'package:test/test.dart';

void main() {
  test('caller traces retain panic frames without changing fatal traces', () {
    const uri = 'package:goruntime/src/panic.dart';
    final trace = StackTrace.fromString(
      '#0 new GoPanic ($uri:62:9)\n'
      '#1 new GoPanic (package:other/panic.dart:62:9)',
    );
    expect($goStack(trace), isEmpty);
    final frames = $goStack(trace, panicFrames: true);
    expect(frames, hasLength(1));
    expect(frames.single.function, 'runtime.gopanic');
    expect(frames.single.file, uri);
    expect(frames.single.line, 62);
  });
  test('initialization frames retain the actual runtime source location', () {
    const uri = 'package:goruntime/src/init.dart';
    final frames = $goStack(
      StackTrace.fromString(
        '#0 _InitSequence.resume ($uri:37:9)\n'
        '#1 _InitValue.resume ($uri:70:9)\n'
        '#2 _InitSequence.resume (package:other/init.dart:37:9)',
      ),
    );
    expect(frames, hasLength(1));
    expect(frames.single.function, 'runtime.doInit');
    expect(frames.single.file, uri);
    expect(frames.single.line, 37);
  });
  for (final uri in <String>[
    'file:///tmp/trace.dart',
    r'C:\project\trace.dart',
    'package:fixture/trace.dart',
  ]) {
    test('map VM and AOT trace locations in $uri', () {
      $srcmap(<String>[uri, '0|fixture.go|main.work|6\n1+7']);
      for (final location in <String>['$uri:1:23', '$uri:1']) {
        final frames = $goStack(StackTrace.fromString('#0 work ($location)'));
        expect(frames, hasLength(1));
        expect(frames.single.function, 'main.work');
        expect(frames.single.file, 'fixture.go');
        expect(frames.single.line, 7);
      }
    });
  }
  test('omit unknown and malformed locations', () {
    expect(
      $goStack(
        StackTrace.fromString(
          '#0 work (file:///unknown.dart:1)\n#1 work (trace.dart:unknown)',
        ),
      ),
      isEmpty,
    );
  });
  test('source positions share the function entry', () {
    $srcmap(<String>[
      'package:fixture/entries.dart',
      '1|entries.go|fixture.first|3;4|entries.go|fixture.second|20\n1+5,2+3,2+c',
    ]);
    final first = $recordPC(GoString.fromString('fixture.first|entries.go|5'));
    final next = $recordPC(GoString.fromString('fixture.first|entries.go|8'));
    final second = $recordPC(
      GoString.fromString('fixture.second|entries.go|21'),
    );
    expect(first, isNot(next));
    expect($pcEntry(first), $pcEntry(next));
    expect($pcEntry(second), isNot($pcEntry(first)));
    expect($pcInfo(first).toDartString(), 'fixture.first|entries.go|5');
    expect($pcInfo(next).toDartString(), 'fixture.first|entries.go|8');
    expect(
      $pcInfo($pcEntry(first)).toDartString(),
      'fixture.first|entries.go|3',
    );
    expect($pcEntry(0), 0);
    expect($pcEntry(-1), 0);
    final sameLine = $recordPC(
      GoString.fromString('fixture.first|entries.go|3'),
    );
    expect(sameLine, isNot($pcEntry(sameLine)));
    expect($pcInfo(sameLine), $pcInfo($pcEntry(sameLine)));
  });
  test('generic function entries retain distinct identities and aliases', () {
    final info = GoString.fromString('fixture.G[...].M|generic.go|12');
    final first = $functionValue(() => 1, info, 'fixture.G[int].M');
    final again = $functionValue(() => 2, info, 'fixture.G[int].M');
    final other = $functionValue(() => 3, info, 'fixture.G[string].M');
    final firstPC = $functionPC(first)!;
    final otherPC = $functionPC(other)!;
    expect($functionPC(again), firstPC);
    expect(otherPC, isNot(firstPC));
    expect($pcEntry(firstPC), firstPC);
    expect($pcEntry(otherPC), otherPC);
    expect($pcInfo(firstPC), info);
    expect($pcInfo(otherPC), info);
    final alias = $copyFunctionPC(() => 4, first);
    expect($functionPC(alias), firstPC);
    expect(alias(), 4);
  });

}
