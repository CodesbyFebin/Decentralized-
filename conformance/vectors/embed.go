// Package vectors embeds the dh/v1 conformance vectors so binaries can run
// the suite without the source tree.
package vectors

import _ "embed"

//go:embed dh-v1.json
var DHv1 []byte
