// A record with exactly 20 fields stops `dart compile wasm`.
//
// dart2wasm builds 1 implementation class per record shape and gives it a
// hashCode getter. That getter calls Object.hash with the shape id and with
// each field, thus 21 arguments for a record of 20 fields. Object.hash holds
// only 20 positional parameters, object1 to object20.
//
// The type flow analysis of pkg/vm then applies the summary of Object.hash to
// 21 positional arguments. Statement 21 of that summary is a Call and not a
// Parameter, thus the cast on summary.dart line 1094 fails.
//
// The guard sits in pkg/dart2wasm/lib/record_class_generator.dart, in
// _generateHashCode: `if (fields.length <= 20)`. The bound must be 19, because
// the argument list also carries the shape id.
//
// A record of 19 fields passes. A record of 21 fields passes too, because
// dart2wasm falls back to Object.hashAll above 20 fields.
//
// The record needs no use at run time. The generated hashCode getter carries a
// wasm:entry-point pragma, thus the analysis reads it even from dead code.
//
// The summary of Object.hash exceeds largeSummarySize, which is 300. After
// maxDirectInvocationsPerSelector, which is 10, distinct invocations the
// analysis replaces each further call with raw arguments and the crash stops.
// A larger program therefore hides the fault when other record shapes hash
// first, and a small unrelated edit can bring the fault back.
void main() {
  print((1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20));
}
