import 'dart:async';

Future<String> greeting(String name) async => 'Hello $name';
Future<String> failure() async => throw StateError('service failure');

int cancellations = 0;
Stream<int> numbers() async* {
  try {
    yield 3;
    yield 5;
    yield 8;
  } finally {
    cancellations++;
  }
}

Stream<int> broken() async* {
  yield 1;
  throw StateError('stream failure');
}
