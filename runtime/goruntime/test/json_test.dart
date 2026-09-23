import 'package:goruntime/goruntime.dart';
import 'package:test/test.dart';

void main() {
  final TypeDesc record = TypeDesc(
    'example.Record',
    'struct',
    const <String, Function>{},
    fields: <FieldDesc>[
      FieldDesc(
        'Value',
        'json:"value,omitempty"',
        true,
        () => $typeFloat32,
        (Object? value) => value,
        (Object? value, Object? field) {},
      ),
    ],
  );

  test('float32 signed zeros use the JSON empty-value rule', () {
    for (final int bits in <int>[0, 0x80000000]) {
      final Slice<GoInt> result = $jsonMarshal(
        Iface(record, GoFloat32.fromBits(bits)),
      );
      expect(result.isNil, isFalse);
      expect(result.len, 2);
      expect(result[0], $gi(123));
      expect(result[1], $gi(125));
    }
  });

  test('nonzero float32 values retain the JSON fallback', () {
    final Slice<GoInt> result = $jsonMarshal(
      Iface(record, GoFloat32.fromDouble(1)),
    );
    expect(result.isNil, isTrue);
  });
}
