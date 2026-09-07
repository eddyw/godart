"""Compare Compass v2 with the original sample at 2 sizes and 2 themes."""

import json
import shutil
import re
import subprocess
import tempfile
from pathlib import Path


COMMIT = "463e365e4842f252ffab9c6198594a504d69469f"
APP = Path(__file__).resolve().parents[1]


def run(arguments, directory):
    subprocess.run(arguments, cwd=directory, check=True)


def prepare_reference(source, reference):
    shutil.copytree(source / "compass_app/app", reference)
    pubspec = reference / "pubspec.yaml"
    text = pubspec.read_text().replace("resolution: workspace\n", "")
    text = text.replace(
        "  analysis_defaults:\n    path: ../../analysis_defaults\n",
        "  flutter_lints: ^6.0.0\n",
    ).replace(
        "dev_dependencies:\n",
        "dev_dependencies:\n  file: ^7.0.1\n  flutter_cache_manager: ^3.4.2\n",
    ).replace(
        "    - assets/user.jpg",
        "    - assets/user.jpg\n    - assets/fonts/\n    - assets/photos/",
    )
    pubspec.write_text(text)
    (reference / "analysis_options.yaml").write_text(
        "include: package:flutter_lints/flutter.yaml\n"
    )
    shutil.copytree(APP / "assets/fonts", reference / "assets/fonts")
    shutil.copytree(APP / "assets/photos", reference / "assets/photos")
    shutil.copyfile(APP / "lib/photos.dart", reference / "lib/photos.dart")
    for name in ["fonts.dart", "images.dart"]:
        (reference / "test" / name).write_text(
            (APP / "test" / name).read_text().replace(
                "package:compass_v2/photos.dart", "package:compass_app/photos.dart"
            )
        )

    flow = (APP / "test/flow.dart").read_text().replace(
        "package:material_ui/material_ui.dart", "package:flutter/material.dart"
    )
    flow = "\n".join(
        f"import 'package:compass_app/{name}';"
        for name in [
            "ui/core/ui/home_button.dart",
            "ui/core/ui/back_button.dart",
            "ui/results/widgets/result_card.dart",
        ]
    ) + "\n" + flow
    flow = flow.replace(
        "find.byKey(const ValueKey('home-button'))", "find.byType(HomeButton)"
    ).replace(
        "find.byKey(const ValueKey('back-button'))", "find.byType(CustomBackButton)"
    ).replace(
        "find.byKey(const ValueKey('amalfi-coast'))", "find.byType(ResultCard).first"
    )
    endpoint = "  await capture?.call('booking');"
    flow = flow[: flow.index(endpoint)] + endpoint + "\n}\n"
    flow = re.sub(
        r"if\s*\(find\.byType\(CircularProgressIndicator\)\.evaluate\(\)\.isEmpty\)",
        "if (i > 9 && find.byType(CircularProgressIndicator).evaluate().isEmpty)",
        flow,
    )
    (reference / "test/flow.dart").write_text(flow)

    capture = (APP / "tool/capture_test.dart").read_text().replace(
        "package:material_ui/material_ui.dart", "package:flutter/material.dart"
    ).replace(
        "import 'package:compass_v2/main.dart';",
        """import 'package:compass_app/main.dart';
import 'package:compass_app/config/dependencies.dart';
import 'package:compass_app/data/repositories/auth/auth_repository.dart';
import 'package:provider/provider.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:flutter/services.dart';
import '../testing/fakes/repositories/fake_auth_repository.dart';""",
    ).replace(
        "installCompass();", "GoogleFonts.config.allowRuntimeFetching = false;"
    ).replace(
        "installImageFixture();",
        """installImageFixture();
        await tester.runAsync(() async {
          await rootBundle.loadString('assets/destinations.json');
          await rootBundle.loadString('assets/activities.json');
        });""",
    ).replace(
        "child: const CompassApp()",
        """child: MultiProvider(
          providers: [...providersLocal,
            ChangeNotifierProvider<AuthRepository>(create: (_) => FakeAuthRepository())],
          child: const MainApp())""",
    )
    (reference / "tool").mkdir(exist_ok=True)
    (reference / "tool/capture_test.dart").write_text(capture)


def main():
    run(["flutter", "test", "tool/capture_test.dart", "--reporter", "expanded"], APP)
    with tempfile.TemporaryDirectory(prefix="compass-v2-reference-") as directory:
        temporary = Path(directory)
        source = temporary / "source"
        run(
            ["git", "clone", "--filter=blob:none", "--no-checkout", "--depth=1",
             "https://github.com/flutter/samples.git", str(source)],
            temporary,
        )
        run(["git", "sparse-checkout", "set", "compass_app"], source)
        run(["git", "fetch", "--depth=1", "origin", COMMIT], source)
        run(["git", "checkout", "--detach", COMMIT], source)
        reference = temporary / "reference"
        prepare_reference(source, reference)
        run(["flutter", "pub", "get"], reference)
        run(["flutter", "test", "tool/capture_test.dart", "--reporter", "expanded"], reference)
        captures = sorted((APP / "doc/capture").glob("*.png"))
        if len(captures) != 32:
            raise RuntimeError(f"Expected 32 captures, received {len(captures)}")
        different = [
            capture.name for capture in captures
            if capture.read_bytes() != (reference / "doc/capture" / capture.name).read_bytes()
        ]
        shutil.copytree(reference / "doc/capture", APP / "doc/reference", dirs_exist_ok=True)
        report = {
            "upstream": COMMIT,
            "captures": len(captures),
            "equal": len(captures) - len(different),
            "different": different,
            "images": "Original login and continent photographs. The same fixture for unavailable catalog images.",
        }
        (APP / "doc/parity.json").write_text(json.dumps(report, indent=2) + "\n")
        if different:
            raise RuntimeError(f"Different captures: {', '.join(different)}")
        print("All 32 captures match the original sample byte for byte.")


if __name__ == "__main__":
    main()
