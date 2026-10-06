import 'dart:io';

import 'package:goruntime/goruntime.dart';
import 'package:test/test.dart';

void main() {
  test('Unlink preserves directory and missing-path errors', () {
    final result = Process.runSync(Platform.resolvedExecutable, [
      'run',
      '--verbosity=error',
      'test/fixtures/unlink_native_errno.dart',
    ]);
    expect(result.exitCode, 0, reason: '${result.stdout}${result.stderr}');
  }, skip: !(Platform.isLinux || Platform.isMacOS));

  test('Native open preserves errors on the first failure', () {
    final result = Process.runSync(Platform.resolvedExecutable, [
      'test/fixtures/native_open_errors.dart',
    ]);
    expect(result.exitCode, 0, reason: '${result.stdout}${result.stderr}');
  }, skip: !(Platform.isLinux || Platform.isMacOS));

  test('Rename preserves raw source and destination bytes', () {
    final root = Directory.systemTemp.createTempSync('gart-rename-');
    addTearDown(() => root.deleteSync(recursive: true));
    final from = GoString.fromBytes([
      ...GoString.fromString('${root.path}/source-').bytes,
      0xff,
    ]);
    final to = GoString.fromBytes([
      ...GoString.fromString('${root.path}/target-').bytes,
      0xfe,
    ]);
    File.fromRawPath(from.bytes).writeAsStringSync('source');
    File.fromRawPath(to.bytes).writeAsStringSync('target');
    final sourceDecoy = File('${root.path}/source-\uFFFD')
      ..writeAsStringSync('source decoy');
    final targetDecoy = File('${root.path}/target-\uFFFD')
      ..writeAsStringSync('target decoy');
    expect($nativeRename(from, to), 0);
    expect(File.fromRawPath(from.bytes).existsSync(), isFalse);
    expect(File.fromRawPath(to.bytes).readAsStringSync(), 'source');
    expect(sourceDecoy.readAsStringSync(), 'source decoy');
    expect(targetDecoy.readAsStringSync(), 'target decoy');
    expect($nativeRename(from, to), 2);
    expect($nativeRename(to, GoString.fromBytes([...from.bytes, 0])), 22);
    expect(File.fromRawPath(to.bytes).readAsStringSync(), 'source');
  }, skip: !Platform.isLinux);

  test('Remove preserves raw names and symbolic link targets', () {
    final root = Directory.systemTemp.createTempSync('gart-remove-');
    addTearDown(() => root.deleteSync(recursive: true));
    final raw = Uint8List.fromList([
      ...GoString.fromString('${root.path}/raw-').bytes,
      0xff,
    ]);
    final path = GoString.fromBytes(raw);
    final decoy = File('${root.path}/raw-\uFFFD')..writeAsStringSync('keep');
    File.fromRawPath(raw).writeAsStringSync('remove');
    expect($remove(path).$1, 0);
    expect(File.fromRawPath(raw).existsSync(), isFalse);
    expect(decoy.readAsStringSync(), 'keep');
    Directory.fromRawPath(raw).createSync();
    expect($remove(path).$1, 0);
    expect(Directory.fromRawPath(raw).existsSync(), isFalse);
    Link.fromRawPath(raw).createSync(decoy.path);
    expect($remove(path).$1, 0);
    expect(Link.fromRawPath(raw).existsSync(), isFalse);
    expect(decoy.readAsStringSync(), 'keep');
    Directory.fromRawPath(raw).createSync();
    File.fromRawPath(Uint8List.fromList([...raw, 47, 120]))
        .writeAsStringSync('keep');
    expect($remove(path).$1, 39);
    expect(Directory.fromRawPath(raw).existsSync(), isTrue);
    expect($remove(GoString.fromString('${decoy.path}\x00suffix')).$1, 22);
    expect(decoy.readAsStringSync(), 'keep');
  }, skip: !Platform.isLinux);

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

  test('Native path attributes preserve raw bytes and file identity', () {
    final root = Directory.systemTemp.createTempSync('gart-pathattrs-');
    addTearDown(() => root.deleteSync(recursive: true));
    final raw = Uint8List.fromList([
      ...GoString.fromString('${root.path}/raw-').bytes,
      0xff,
    ]);
    File.fromRawPath(raw).writeAsStringSync('raw');
    final path = GoString.fromBytes(raw);
    final decoded = File('${root.path}/raw-\uFFFD')
      ..writeAsStringSync('decoded');
    final decodedPath = GoString.fromString(decoded.path);
    final before = $nativeStat(decodedPath, true);
    expect(before.$2, 0);
    expect($nativeChmod(path, 0x100), 0);
    final stat = $nativeStat(path, true);
    expect(stat.$2, 0);
    expect(stat.$1[1] & 0x1ff, 0x100);
    expect($nativeChown(path, stat.$1[7], stat.$1[8], true), 0);
    expect($nativeChown(path, -1, -1, false), 0);
    final after = $nativeStat(decodedPath, true);
    expect(after.$2, 0);
    expect(after.$1[1], before.$1[1]);
    expect(after.$1[7], before.$1[7]);
    expect(after.$1[8], before.$1[8]);
  }, skip: !Platform.isLinux);

  test('RemoveAll rejects a NUL path without removal', () {
    final root = Directory.systemTemp.createTempSync('gart-removeall-');
    addTearDown(() {
      if (root.existsSync()) root.deleteSync(recursive: true);
    });
    final file = File('${root.path}/file')..writeAsStringSync('keep');
    final result = $removeAll(GoString.fromString('${root.path}\x00suffix'));
    expect(result.$3, 22);
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
      expect(result.$3, 0, reason: result.toString());
      expect(tree.existsSync(), isFalse);
      expect(outside.readAsStringSync(), 'keep');
    },
    skip: !Platform.isLinux,
  );

  test('RemoveAll closes directory descriptors after repeated traversal', () {
    final result = Process.runSync(Platform.resolvedExecutable, [
      'run',
      '--verbosity=error',
      'test/fixtures/removeall_descriptors.dart',
    ]);
    expect(result.exitCode, 0, reason: '${result.stdout}${result.stderr}');
  }, skip: !(Platform.isLinux || Platform.isMacOS));
}
