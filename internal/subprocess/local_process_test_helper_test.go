//go:build !windows

package subprocess

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const shellInitResponse = `read -r init_line
request_id=$(printf '%s\n' "$init_line" | sed -n 's/.*"request_id":"\([^"]*\)".*/\1/p')
printf '{"type":"control_response","response":{"subtype":"success","request_id":"%s","response":{}}}\n' "$request_id"
`

func createTransportTempScript(t *testing.T, script, extension string) string {
	t.Helper()
	versionLine := `if [ "$1" = "-v" ]; then echo "3.0.0"; exit 0; fi` + "\n"
	if !strings.Contains(script, "# NO_HANDSHAKE") {
		script = strings.Replace(script, versionLine, versionLine+shellInitResponse, 1)
	}
	path := filepath.Join(t.TempDir(), "mock-claude"+extension)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
