package release_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestDefaultDaemonVersion(t *testing.T) {
	for _, tt := range []struct {
		name, tag, want string
	}{
		{"untagged build", "", "0.0.0.gc0abf6bfff42"},
		{"tagged build", "v0.8.0", "v0.8.0"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			manifest := filepath.Join(t.TempDir(), "manifest.yaml")
			contents := "daemon_tag: \"" + tt.tag + "\"\ndaemon_commit: \"c0abf6bfff429c44bf4949cdfe72dcbbb2019b46\"\n"
			if err := os.WriteFile(manifest, []byte(contents), 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("bash", "-c", `source ./common.sh && default_daemon_version`)
			cmd.Env = append(os.Environ(), "MANIFEST_PATH="+manifest)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("default_daemon_version: %v: %s", err, output)
			}
			got := strings.TrimSpace(string(output))
			if got != tt.want {
				t.Fatalf("version = %q, want %q", got, tt.want)
			}
			if !regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+`).MatchString(got) {
				t.Fatalf("client version must start with a numeric version: %q", got)
			}
		})
	}
}
