package release_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageMetadataUsesReleaseVersionAndAudioDependencies(t *testing.T) {
	for _, kind := range []string{"aur", "homebrew"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			manifest := filepath.Join(dir, "manifest.yaml")
			if err := os.WriteFile(manifest, []byte("lazyspotify_version: \"0.5.2\"\ndaemon_tag: \"v0.8.0\"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			archive := filepath.Join(dir, "artifact.tar.gz")
			contents := []byte("test artifact for checksum rendering")
			if err := os.WriteFile(archive, contents, 0600); err != nil {
				t.Fatal(err)
			}
			checksum := fmt.Sprintf("%x", sha256.Sum256(contents))
			var cmd *exec.Cmd
			var outputs []string
			if kind == "aur" {
				outputs = []string{filepath.Join(dir, "PKGBUILD"), filepath.Join(dir, ".SRCINFO")}
				cmd = exec.Command("bash", "./render-aur-metadata.sh", "--archive", archive, "--pkgbuild-output", outputs[0], "--srcinfo-output", outputs[1])
			} else {
				outputs = []string{filepath.Join(dir, "lazyspotify.rb")}
				cmd = exec.Command("bash", "./render-homebrew-formula.sh", "--source-bundle", archive, "--output", outputs[0], "--commit", "test-commit", "--build-date", "2026-10-15T00:00:00Z")
			}
			cmd.Env = append(os.Environ(), "MANIFEST_PATH="+manifest, "GITHUB_REPOSITORY=dubeyKartikay/lazyspotify")
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("render: %v: %s", err, output)
			}
			for _, file := range outputs {
				data, err := os.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				text := string(data)
				for _, want := range []string{"0.5.2", "/releases/download/v0.5.2/", checksum, "mpg123"} {
					if !strings.Contains(text, want) {
						t.Errorf("%s missing %q", filepath.Base(file), want)
					}
				}
				if kind == "homebrew" {
					if !strings.Contains(text, "go-librespot.version=v0.8.0") || strings.Contains(text, "@DAEMON_VERSION@") {
						t.Error("Homebrew daemon version was not rendered")
					}
					for _, lib := range []string{"flac", "libogg", "libvorbis", "mpg123"} {
						if !strings.Contains(text, "depends_on \""+lib+"\"\n") {
							t.Errorf("%s must be a runtime dependency", lib)
						}
					}
				}
			}
		})
	}
}
