import 'slice.dart';
import 'string.dart';

(Slice<GoString>, int, GoString) $verifySystemCertificates(
  Slice<GoString> certificates,
  GoString name,
  double date,
  bool hasDate,
) => (
  Slice<GoString>.nil(),
  -1,
  GoString.ascii('platform certificate verification is unavailable'),
);
