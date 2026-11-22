//go:build tinymodel

package library

import "github.com/kheif/quiver/internal/embed"

// The end-to-end tests build quiver with this tag to run the local engine
// against the tiny model with random weights that embedtest writes. No other
// build has it. A size and hashes of zero are not checked.
func init() {
	models["tiny"] = ModelSpec{Name: "tiny", Config: embed.Config{Vocab: 64, Hidden: 8, Layers: 2, Heads: 2, Inter: 16, MaxPos: 512}, MinScore: -1}
}
