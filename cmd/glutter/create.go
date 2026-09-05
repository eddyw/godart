package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// platforms maps the name of a device to the platform of flutter create.
var platforms = map[string]string{
	"macos":   "macos",
	"linux":   "linux",
	"windows": "windows",
	"android": "android",
	"ios":     "ios",
	"web":     "web",
	"chrome":  "web",
}

// create writes the platform runner of the application. git holds no such
// directory, because flutter test needs none and each file comes from flutter
// create.
func (a *app) create(args []string) error {
	name := "macos"
	if d := deviceOf(args); d != "" {
		name = d
	}
	for _, s := range args {
		if !strings.HasPrefix(s, "-") {
			name = s
		}
	}
	platform, ok := platforms[name]
	if !ok {
		return fmt.Errorf("no platform of the name %s. Write macos, ios, android, linux, windows, or web", name)
	}
	return a.runner(platform)
}

// runner writes the platform runner when the directory of the platform is
// absent.
func (a *app) runner(device string) error {
	platform, ok := platforms[device]
	if !ok {
		return nil
	}
	if _, err := os.Stat(filepath.Join(a.dir, platform)); err == nil {
		return nil
	}
	fmt.Printf("glutter: the %s runner is absent, thus flutter create writes it\n", platform)
	readme := filepath.Join(a.dir, "README.md")
	held, _ := os.ReadFile(readme)
	if err := run1(a.dir, "flutter", "create", "--platforms="+platform, "."); err != nil {
		return err
	}
	// flutter create writes a template test that names a widget of its own,
	// and 3 files of an editor. The application wants none of them.
	for _, junk := range []string{".idea", ".metadata", "test/widget_test.dart"} {
		os.RemoveAll(filepath.Join(a.dir, junk))
	}
	if held != nil {
		if err := os.WriteFile(readme, held, 0o644); err != nil {
			return err
		}
	}
	return a.permissions(platform)
}

// permissions patches the 2 files that hold back a network call. flutter
// create writes neither 1, thus an application that reads a server reaches
// none.
func (a *app) permissions(platform string) error {
	switch platform {
	case "macos":
		for _, name := range []string{
			"macos/Runner/DebugProfile.entitlements",
			"macos/Runner/Release.entitlements",
		} {
			p := filepath.Join(a.dir, name)
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			s := string(b)
			if strings.Contains(s, "network.client") {
				continue
			}
			s = strings.Replace(s,
				"<key>com.apple.security.app-sandbox</key>",
				"<key>com.apple.security.network.client</key>\n\t<true/>\n\t<key>com.apple.security.app-sandbox</key>", 1)
			if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
				return err
			}
		}
	case "android":
		p := filepath.Join(a.dir, "android/app/src/main/AndroidManifest.xml")
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		s := string(b)
		if strings.Contains(s, "android.permission.INTERNET") {
			return nil
		}
		s = strings.Replace(s, "    <application",
			"    <uses-permission android:name=\"android.permission.INTERNET\"/>\n    <application", 1)
		return os.WriteFile(p, []byte(s), 0o644)
	}
	return nil
}
