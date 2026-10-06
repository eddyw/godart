import 'dart:io';

import 'package:goruntime/src/identity.dart';
import 'package:goruntime/src/native_errno.dart';
import 'package:test/test.dart';

void main() {
  test('native signal queries retain errors and recover', () {
    for (var attempt = 0; attempt < 32; attempt++) {
      final absent = nativeKill(2147483647, 0);
      expect(absent.value, -1);
      expect(absent.error, 3);
      final invalid = nativeKill(pid, -1);
      expect(invalid.value, -1);
      expect(invalid.error, 22);
      final self = nativeKill(pid, 0);
      expect(self.value, 0);
      expect(self.error, 0);
      expect($sendSignal(2147483647, 0), 3);
      expect($sendSignal(pid, -1), 22);
      expect($sendSignal(pid, 0), 0);
    }
  }, skip: !Platform.isLinux && !Platform.isMacOS);
}
