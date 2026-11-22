//go:build tinymodel

package library

import (
	"crypto/tls"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/kheif/quiver/internal/embed"
)

// The end-to-end tests build quiver with this tag to run the local engine
// against the tiny model with random weights that embedtest writes. No other
// build has it. A size and hashes of zero are not checked.
//
// With QUIVER_TINY_MODEL_PIN, "address size sha256 tokenizer-size
// tokenizer-sha256" separated by spaces, the model can also be downloaded
// from a fake host that the test runs, whose certificate this build trusts.
// The production build has neither the variable nor the trust.
func init() {
	spec := ModelSpec{Name: "tiny", Config: embed.Config{Vocab: 64, Hidden: 8, Layers: 2, Heads: 2, Inter: 16, MaxPos: 512}, MinScore: -1}
	if f := strings.Fields(os.Getenv("QUIVER_TINY_MODEL_PIN")); len(f) == 5 {
		spec.BaseURL = f[0]
		spec.Size, _ = strconv.ParseInt(f[1], 10, 64)
		spec.SHA256 = f[2]
		spec.TokenizerSize, _ = strconv.ParseInt(f[3], 10, 64)
		spec.TokenizerSHA256 = f[4]
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // the fake host of a test
		modelTransport = t
	}
	models["tiny"] = spec
}
