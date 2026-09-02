import 'package:integration_test/integration_test.dart';

import '../test/skip_test.dart' as measurement;

/// The profile run of the measurement of test/skip_test.dart.
///
/// A widget test runs in debug mode: the Dart VM interprets the code, and each
/// assert of the framework runs. A profile build drops both, thus only this
/// run gives a number that a real application sees. Read decision 253.
void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  measurement.main();
}
