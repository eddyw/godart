import 'package:goruntime/goruntime.dart';
import 'package:test/test.dart';

void main() {
  for (final uri in <String>[
    'file:///tmp/trace.dart',
    r'C:\project\trace.dart',
    'package:fixture/trace.dart',
  ]) {
    test('map VM and AOT trace locations in $uri', () {
      $srcmap(<String>[uri, '0|fixture.go|main.work\n1+7']);
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
}
