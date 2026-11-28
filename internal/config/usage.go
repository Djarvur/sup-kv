package config

import (
	"fmt"
	"io"
	"os"
)

// envOnlyBlock is the usage footer for the five knobs with no flag twin: the
// structured-logging pair (D-06/D-07) and the metrics certificate triple
// (D-02). These exact names are the OBSV-02 usage contract.
const envOnlyBlock = `Environment-only (no flag):
  SUPKV_LOG_LEVEL          debug, info, warn or error (default info)
  SUPKV_LOG_FORMAT         json or text (default json)
  SUPKV_METRICS_CERT       server certificate path (all three or none)
  SUPKV_METRICS_KEY        server key path (all three or none)
  SUPKV_METRICS_CLIENT_CA  client CA path (all three or none)
`

// Usage renders the flag table [Parse] itself registers — so the
// `-flag ... (env SUPKV_X)` and `(default ...)` lines cannot drift from the
// parser — followed by the env-only block naming the five flagless knobs.
//
// The defaults resolve from the process environment exactly as Parse resolves
// them, so `SUPKV_MAX_CLOCK_LEAD=90s --help` shows `(default 1m30s)`: the help
// text reports the effective default, never a second hardcoded copy.
func Usage(w io.Writer) {
	var cfg Config
	fs := newFlagSet(&cfg, &resolver{getenv: os.Getenv})
	fs.SetOutput(w)
	fs.PrintDefaults()
	_, _ = fmt.Fprint(w, envOnlyBlock)
}
