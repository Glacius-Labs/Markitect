// Package markitect carries the source of the image the native app's chats run
// in, so that the app can build that image on a machine with Docker and no
// checkout. It sits at the repository's root because an embedded file must lie
// below the package that embeds it. Only the native app imports it. It embeds
// the files as they are when the app is built, so a release comes from a clean
// checkout.
package markitect

import "embed"

// ImageSource is the build context of packaging/runtime/Dockerfile.
//
//go:embed go.mod go.sum all:internal cmd/markitect cmd/markitect-agentd packaging/runtime/Dockerfile
var ImageSource embed.FS
