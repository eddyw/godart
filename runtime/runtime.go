// Package runtime holds the Dart runtime shipped with the compiler.
package runtime

import "embed"

// Files contains the runtime package, license, and dependency manifest.
//
//go:embed goruntime/lib goruntime/pubspec.yaml goruntime/LICENSE
var Files embed.FS
