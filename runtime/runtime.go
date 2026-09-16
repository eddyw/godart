// Package runtime holds the Dart runtime shipped with the compiler.
package runtime

import "embed"

// Files contains the runtime package and its dependency manifest.
//
//go:embed goruntime/lib goruntime/pubspec.yaml
var Files embed.FS
