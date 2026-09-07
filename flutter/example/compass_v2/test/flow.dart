import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

Future<void> tapAndSettle(WidgetTester tester, Finder finder) async {
  await tester.tap(finder);
  for (var i = 0; i < 100; i++) {
    await tester.runAsync(
      () => Future<void>.delayed(const Duration(milliseconds: 10)),
    );
    await tester.pump();
    if (find.byType(CircularProgressIndicator).evaluate().isEmpty) break;
  }
  await tester.pumpAndSettle();
  expect(tester.takeException(), isNull);
}

Future<void> tripFlow(
  WidgetTester tester, {
  Future<void> Function(String)? capture,
}) async {
  await tester.pumpAndSettle();
  expect(find.text('Login'), findsOneWidget);
  await capture?.call('login');
  await tapAndSettle(tester, find.text('Login'));
  expect(find.text("Sofie's Trips"), findsOneWidget);
  expect(find.text('Alaska, North America'), findsOneWidget);
  await capture?.call('home');
  await tapAndSettle(tester, find.byKey(const ValueKey('booking-button')));
  expect(find.text('Search destination'), findsOneWidget);
  final submit = find.byKey(const ValueKey('submit-button'));
  expect(
    tester
        .widget<FilledButton>(
          find.descendant(
            of: submit,
            matching: find.byType(FilledButton),
            matchRoot: true,
          ),
        )
        .onPressed,
    isNull,
  );
  await capture?.call('search');
  await tapAndSettle(tester, find.byKey(const ValueKey('Europe')));
  await tapAndSettle(tester, find.byKey(const ValueKey('Europe')));
  await tapAndSettle(tester, find.byKey(const ValueKey('Europe')));
  final overlay = tester.widget<AnimatedOpacity>(
    find.descendant(
      of: find.byKey(const ValueKey('Asia')),
      matching: find.byType(AnimatedOpacity),
    ),
  );
  expect(overlay.opacity, 0.7);
  final overlayBox = find.descendant(
    of: find.byKey(const ValueKey('Asia')),
    matching: find.byType(AnimatedOpacity),
  );
  expect(tester.getSize(overlayBox), const Size(140, 140));
  await tapAndSettle(tester, find.byKey(const ValueKey('remove-guests')));
  await tapAndSettle(tester, find.byKey(const ValueKey('add-guests')));
  await tapAndSettle(tester, find.text('Add Dates'));
  expect(find.byType(DateRangePickerDialog), findsOneWidget);
  await capture?.call('dates');
  await tapAndSettle(tester, find.byTooltip('Switch to input'));
  final start = DateTime.now().add(const Duration(days: 2));
  final end = start.add(const Duration(days: 3));
  String date(DateTime d) => '${d.month}/${d.day}/${d.year}';
  await tester.enterText(find.byType(TextField).at(0), date(start));
  await tester.enterText(find.byType(TextField).at(1), date(end));
  await tapAndSettle(tester, find.text('OK'));
  await capture?.call('search-filled');
  await tapAndSettle(tester, submit);
  expect(find.text('AMALFI COAST'), findsOneWidget);
  await capture?.call('results');
  await tapAndSettle(tester, find.byKey(const ValueKey('amalfi-coast')));
  expect(find.text('Activities'), findsOneWidget);
  expect(find.text('0 selected'), findsOneWidget);
  final selection = find.byWidgetPredicate(
    (widget) =>
        widget.key is ValueKey<String> &&
        ((widget.key! as ValueKey<String>).value).endsWith('-checkbox'),
  );
  await tapAndSettle(tester, selection.first);
  expect(find.text('1 selected'), findsOneWidget);
  await capture?.call('activities');
  await tapAndSettle(tester, find.byKey(const ValueKey('back-button')));
  expect(find.text('AMALFI COAST'), findsOneWidget);
  await tapAndSettle(tester, find.byKey(const ValueKey('amalfi-coast')));
  expect(find.text('0 selected'), findsOneWidget);
  await tapAndSettle(tester, selection.first);
  await tapAndSettle(tester, find.byKey(const ValueKey('confirm-button')));
  expect(find.text('Amalfi Coast'), findsOneWidget);
  expect(find.text('Your chosen activities'), findsOneWidget);
  await capture?.call('booking');
  await tapAndSettle(tester, find.byKey(const ValueKey('home-button')));
  expect(find.text('Amalfi Coast, Europe'), findsOneWidget);
  await tapAndSettle(tester, find.text('Amalfi Coast, Europe'));
  expect(find.text('Amalfi Coast'), findsOneWidget);
  await tapAndSettle(tester, find.byKey(const ValueKey('home-button')));
  await tester.drag(find.text('Amalfi Coast, Europe'), const Offset(-1100, 0));
  await tester.pumpAndSettle();
  expect(find.text('Amalfi Coast, Europe'), findsNothing);
  expect(find.text('Booking deleted'), findsOneWidget);
  await tapAndSettle(tester, find.byKey(const ValueKey('logout-button')));
  expect(find.text('Login'), findsOneWidget);
  expect(tester.takeException(), isNull);
}
