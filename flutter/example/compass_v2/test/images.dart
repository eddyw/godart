import 'dart:io' as io;

import 'package:cached_network_image/cached_network_image.dart';
import 'package:compass_v2/photos.dart';
import 'package:file/memory.dart';
import 'package:flutter_cache_manager/flutter_cache_manager.dart';

class FixtureCache extends BaseCacheManager {
  final files = MemoryFileSystem();

  @override
  Stream<FileResponse> getFileStream(
    String url, {
    String? key,
    Map<String, String>? headers,
    bool withProgress = false,
  }) async* {
    final path = bundledPhotos[url] ?? 'assets/user.jpg';
    final file = files.file(path)..parent.createSync(recursive: true);
    file.writeAsBytesSync(io.File(path).readAsBytesSync());
    yield FileInfo(file, FileSource.Cache, DateTime(2100), url);
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

void installImageFixture() {
  CachedNetworkImageProvider.defaultCacheManager = FixtureCache();
}
