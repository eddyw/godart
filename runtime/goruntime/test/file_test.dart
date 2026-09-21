import 'dart:io';

import 'package:goruntime/goruntime.dart';
import 'package:test/test.dart';

void main() {
  test('Native descriptors retain file identity and ownership', () {
    final result = Process.runSync(Platform.resolvedExecutable, [
      'test/fixtures/file_descriptors.dart',
    ]);
    expect(result.exitCode, 0, reason: '${result.stdout}${result.stderr}');
  }, skip: !(Platform.isLinux || Platform.isMacOS));

  test('RemoveAll rejects a NUL path without removal', () {
    final root = Directory.systemTemp.createTempSync('gart-removeall-');
    addTearDown(() {
      if (root.existsSync()) root.deleteSync(recursive: true);
    });
    final file = File('${root.path}/file')..writeAsStringSync('keep');
    final result = $removeAll(GoString.fromString('${root.path}\x00suffix'));
    expect(result.$3.length, greaterThan(0));
    expect(file.readAsStringSync(), 'keep');
  }, skip: !(Platform.isLinux || Platform.isMacOS));

  test(
    'RemoveAll preserves raw directory names and external symlink targets',
    () {
      final root = Directory.systemTemp.createTempSync('gart-removeall-');
      addTearDown(() {
        if (root.existsSync()) root.deleteSync(recursive: true);
      });
      final outside = File('${root.path}/outside')..writeAsStringSync('keep');
      final tree = Directory('${root.path}/tree')..createSync();
      final raw = Uint8List.fromList([
        ...GoString.fromString('${tree.path}/').bytes,
        0xff,
      ]);
      File.fromRawPath(raw).writeAsStringSync('remove');
      Link('${tree.path}/link').createSync(outside.path);
      Link('${tree.path}/broken').createSync('${root.path}/absent');
      final result = $removeAll(GoString.fromString(tree.path));
      expect(result.$3.length, 0, reason: result.toString());
      expect(tree.existsSync(), isFalse);
      expect(outside.readAsStringSync(), 'keep');
    },
    skip: !Platform.isLinux,
  );

  test('RemoveAll closes directory descriptors after repeated traversal', () {
    final result = Process.runSync(Platform.resolvedExecutable, [
      'test/fixtures/removeall_descriptors.dart',
    ]);
    expect(result.exitCode, 0, reason: '${result.stdout}${result.stderr}');
  }, skip: !(Platform.isLinux || Platform.isMacOS));
}
