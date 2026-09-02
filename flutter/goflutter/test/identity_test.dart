import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:goflutter/goflutter.dart';

/// These cases drive the key table with no Go tree. A widget test of an
/// example proves the path from a Go node, and this file proves the rules of
/// the table itself.

void main() {
  testWidgets('1 name gives 1 object of each kind', (
    WidgetTester tester,
  ) async {
    final GoIdentities table = GoIdentities();

    expect(identical(table.focusFor('email'), table.focusFor('email')), isTrue);
    expect(identical(table.scrollFor('rows'), table.scrollFor('rows')), isTrue);
    expect(identical(table.statesFor('send'), table.statesFor('send')), isTrue);
    expect(identical(table.focusFor('email'), table.focusFor('name')), isFalse);

    // Initial reaches the controller of the first read alone.
    expect(table.scrollFor('rows', initial: 40).initialScrollOffset, 0);
    expect(table.scrollFor('later', initial: 40).initialScrollOffset, 40);

    // The kind stands in the identifier, thus 1 name serves the 3 kinds.
    expect(table.ids, <String>[
      'focus:email',
      'focus:name',
      'scroll:later',
      'scroll:rows',
      'states:send',
    ]);
  });

  testWidgets('the sweep disposes the object that no widget holds', (
    WidgetTester tester,
  ) async {
    final GoIdentities table = GoIdentities();
    table.open();
    final FocusNode node = table.focusFor('email');
    final Set<String> read = table.close();

    table.retain(read);
    table.sweep();
    expect(table.ids, <String>['focus:email']);

    table.release(read);
    table.sweep();
    expect(table.ids, isEmpty);
    expect(
      () => node.addListener(() {}),
      throwsA(isA<FlutterError>()),
      reason: 'the sweep must dispose the node',
    );
  });

  testWidgets('the sweep disposes an object that no widget ever held', (
    WidgetTester tester,
  ) async {
    // A build that Flutter throws away reads a key and mounts no widget. The
    // object must not stay in the table.
    final GoIdentities table = GoIdentities();
    final ScrollController lost = table.scrollFor('rows');
    table.sweep();

    expect(table.ids, isEmpty);
    expect(() => lost.addListener(() {}), throwsA(isA<FlutterError>()));
  });

  testWidgets('1 key that reaches 2 widgets gives a loud report', (
    WidgetTester tester,
  ) async {
    final GoIdentities table = GoIdentities();
    table.open();
    table.focusFor('email');
    final Set<String> first = table.close();
    table.open();
    table.focusFor('email');
    final Set<String> second = table.close();

    table.retain(first);
    table.retain(second);
    table.sweep();

    final Object? report = tester.takeException();
    expect(report, isA<StateError>());
    expect(report.toString(), contains('focus:email reaches 2 widgets'));

    // The report names the fault, and it keeps the object. A dispose here
    // would break the 2 widgets that still hold it.
    expect(table.ids, <String>['focus:email']);
    table.release(first);
    table.release(second);
    table.sweep();
    expect(table.ids, isEmpty);
  });

  testWidgets('dropAll releases each object of the table', (
    WidgetTester tester,
  ) async {
    final GoIdentities table = GoIdentities();
    table.open();
    final FocusNode node = table.focusFor('email');
    final WidgetStatesController states = table.statesFor('send');
    table.retain(table.close());

    table.dropAll();

    expect(table.ids, isEmpty);
    expect(() => node.addListener(() {}), throwsA(isA<FlutterError>()));
    expect(() => states.addListener(() {}), throwsA(isA<FlutterError>()));
  });
}
