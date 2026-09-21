import 'dart:io';

import 'package:test/test.dart';

void main() {
  test('Interface metadata releases native descriptors', () {
    final result = Process.runSync(Platform.resolvedExecutable, [
      'test/fixtures/interface_descriptors.dart',
    ]);
    expect(result.exitCode, 0, reason: '${result.stdout}${result.stderr}');
  }, skip: !(Platform.isLinux || Platform.isMacOS));
}
