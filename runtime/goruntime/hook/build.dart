import 'dart:io';

import 'package:code_assets/code_assets.dart';
import 'package:hooks/hooks.dart';
import 'package:native_toolchain_c/native_toolchain_c.dart';

Future<void> main(List<String> args) async {
  await build(args, (input, output) async {
    if (!input.config.buildCodeAssets) return;
    var configured = input;
    final code = input.config.code;
    if (code.targetOS == OS.windows) return;
    if (Platform.isLinux &&
        code.targetOS == OS.linux &&
        code.targetArchitecture == Architecture.current &&
        code.cCompiler == null) {
      final tools = <Uri>[];
      for (final name in ['gcc', 'ld', 'ar']) {
        final result = await Process.run('which', [name]);
        if (result.exitCode != 0) {
          throw ProcessException(name, [], '${result.stderr}', result.exitCode);
        }
        tools.add(
          Uri.file(File('${result.stdout}'.trim()).resolveSymbolicLinksSync()),
        );
      }
      final builder = BuildInputBuilder()..json.addAll(input.json);
      builder.addExtension(
        CodeAssetExtension(
          targetArchitecture: code.targetArchitecture,
          targetOS: code.targetOS,
          linkModePreference: code.linkModePreference,
          sanitizer: code.sanitizer,
          cCompiler: CCompilerConfig(
            compiler: tools[0],
            linker: tools[1],
            archiver: tools[2],
          ),
        ),
      );
      configured = builder.build();
    }
    await CBuilder.library(
      name: 'goruntime_errno',
      assetName: 'src/native_errno.dart',
      sources: ['src/native_errno.c'],
    ).run(input: configured, output: output);
  });
}
