import 'dart:io';

import 'package:flutter/services.dart';

Future<void> loadCompassFonts() async {
  final sdk = Platform.resolvedExecutable.split('/bin/cache/').first;
  final roboto = FontLoader('Roboto');
  for (final face in ['Regular', 'Medium', 'Bold']) {
    final bytes = File(
      '$sdk/bin/cache/artifacts/material_fonts/Roboto-$face.ttf',
    ).readAsBytesSync();
    roboto.addFont(Future.value(ByteData.sublistView(bytes)));
  }
  await roboto.load();
  for (final entry in {
    'Rubik': ['Rubik-Medium', 'Rubik-ExtraBold'],
    'OpenSans': ['OpenSans-Medium'],
  }.entries) {
    final loader = FontLoader(entry.key);
    for (final name in entry.value) {
      loader.addFont(rootBundle.load('assets/fonts/$name.ttf'));
    }
    await loader.load();
  }
  final icons = FontLoader('MaterialIcons')
    ..addFont(rootBundle.load('fonts/MaterialIcons-Regular.otf'));
  await icons.load();
}
