import 'dart:async';
import 'dart:convert';

import 'package:material_ui/material_ui.dart';

import 'host.dart';

/// goStore holds each value of ui.SaveValue while the application runs.
///
/// A restart drops the whole map. An application that needs a value across a
/// restart names shared_preferences, and it answers the 2 store calls in its
/// own handler.
final Map<String, String> goStore = <String, String>{};

/// goHostCall answers each call that flutter/ui/calls.go names.
///
/// goSetCall installs the handler of the application, and that handler gives
/// each name that it does not hold to this call.
Future<String> goHostCall(String name, String arg) async {
  switch (name) {
    case 'dateRange':
      return _dateRange(arg);
    case 'snackBar':
      return _snackBar(arg);
    case 'store.set':
      final List<Object?> pair = jsonDecode(arg) as List<Object?>;
      goStore[pair[0]! as String] = pair[1]! as String;
      return '';
    case 'store.get':
      return goStore[arg] ?? '';
    case 'share':
      throw StateError(
        'the host holds no share sheet. An application that names share_plus '
        'answers the call share in the handler of goSetCall.',
      );
  }
  throw StateError('the host holds no call of the name $name');
}

/// _context gives the context of the Anchor node, and it reports the absent
/// node by its own name.
BuildContext _context(String call) {
  final BuildContext? at = goAnchorContext;
  if (at == null) {
    throw StateError(
      'the call $call needs a ui.Anchor node under the application.',
    );
  }
  return at;
}

/// _dateRange opens the date range dialog of Material.
Future<String> _dateRange(String arg) async {
  final List<int> at = <int>[
    for (final String part in arg.split(',')) int.parse(part),
  ];
  final DateTime now = DateTime.now();
  final DateTime first = at[0] == 0
      ? DateTime(now.year, now.month, now.day)
      : DateTime.fromMillisecondsSinceEpoch(at[0]);
  final DateTime last = at[1] == 0
      ? first.add(const Duration(days: 365))
      : DateTime.fromMillisecondsSinceEpoch(at[1]);
  final DateTimeRange? start = at[2] == 0 || at[3] == 0
      ? null
      : DateTimeRange(
          start: DateTime.fromMillisecondsSinceEpoch(at[2]),
          end: DateTime.fromMillisecondsSinceEpoch(at[3]),
        );
  final DateTimeRange? out = await showDateRangePicker(
    context: _context('dateRange'),
    firstDate: first,
    lastDate: last,
    initialDateRange: start,
  );
  if (out == null) return '0,0';
  return '${out.start.millisecondsSinceEpoch},'
      '${out.end.millisecondsSinceEpoch}';
}

/// _snackBar shows 1 message, and it waits for the end of that message.
Future<String> _snackBar(String arg) {
  final List<Object?> pair = jsonDecode(arg) as List<Object?>;
  final String message = pair[0]! as String;
  final String label = pair[1]! as String;
  final ScaffoldMessengerState? bar = ScaffoldMessenger.maybeOf(
    _context('snackBar'),
  );
  if (bar == null) {
    throw StateError(
      'the Anchor node stands outside a MaterialApp, thus the host holds no '
      'ScaffoldMessenger.',
    );
  }
  final Completer<String> done = Completer<String>();
  final ScaffoldFeatureController<SnackBar, SnackBarClosedReason> shown = bar
      .showSnackBar(
        SnackBar(
          content: Text(message),
          action: label.isEmpty
              ? null
              : SnackBarAction(
                  label: label,
                  onPressed: () {
                    if (!done.isCompleted) done.complete('action');
                  },
                ),
        ),
      );
  shown.closed.then((SnackBarClosedReason reason) {
    if (!done.isCompleted) done.complete('');
  });
  return done.future;
}
