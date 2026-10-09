const bool $webSourceFrames = bool.fromEnvironment('dart.library.js_interop');

// A resume enters a fresh frame. Its finally block removes it before suspension.
final List<SourceFrame> _sourceFrames = [];

final class SourceFrame {
  final String name;
  String file;
  int line;

  SourceFrame(this.name, this.file, this.line);

  void at(String file, int line) {
    this.file = file;
    this.line = line;
  }
}

SourceFrame? $sourceEnter(String name, String file, int line) {
  final frame = SourceFrame(name, file, line);
  _sourceFrames.add(frame);
  return frame;
}

void $sourceLeave(SourceFrame? frame) {
  if (frame != null) _sourceFrames.removeLast();
}

Iterable<(String, String, int)> get $sourceFrames sync* {
  for (final frame in _sourceFrames.reversed) {
    yield (frame.name, frame.file, frame.line);
  }
}

@pragma('vm:prefer-inline')
T $sourceCall<T>(SourceFrame? frame, String file, int line, T Function() call) {
  if (frame == null) return call();
  // Nested argument calls must restore the outer call's source position.
  final previousFile = frame.file;
  final previousLine = frame.line;
  frame.at(file, line);
  try {
    return call();
  } finally {
    frame.at(previousFile, previousLine);
  }
}
