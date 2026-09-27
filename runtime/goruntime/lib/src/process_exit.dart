/// Process termination bypasses Go defer and recover.
final class ProcessExit implements Exception {
  const ProcessExit();
}
