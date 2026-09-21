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

  test('Native timestamps preserve raw path bytes and nanoseconds', () {
    final root = Directory.systemTemp.createTempSync('gart-filetimes-');
    addTearDown(() => root.deleteSync(recursive: true));
    final raw = Uint8List.fromList([
      ...GoString.fromString('${root.path}/raw-').bytes,
      0xff,
    ]);
    File.fromRawPath(raw).writeAsStringSync('value');
    final path = GoString.fromBytes(raw);
    expect($nativeChtimes(path, 1700000000123456789, true, -1, true), 0);
    final stat = $nativeStat(path, true);
    expect(stat.$2, 0);
    expect(stat.$1[12], 1700000000);
    expect(stat.$1[13], 123456789);
    expect(stat.$1[2], -1);
    expect(stat.$1[3], 999999999);
    expect($nativeChtimes(path, 0, false, 789, true), 0);
    final updated = $nativeStat(path, true);
    expect(updated.$2, 0);
    expect(updated.$1[12], stat.$1[12]);
    expect(updated.$1[13], stat.$1[13]);
    expect(updated.$1[2], 0);
    expect(updated.$1[3], 789);
  }, skip: !Platform.isLinux);

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
