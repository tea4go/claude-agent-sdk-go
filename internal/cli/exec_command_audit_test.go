package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRepositoryCommandsUseCentralFactory(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate command audit test")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	allowed := map[string]bool{
		filepath.Join(root, "internal", "cli", "exec_command_unix.go"):    true,
		filepath.Join(root, "internal", "cli", "exec_command_windows.go"): true,
	}
	directCall := "exec." + "Command"

	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			if info.Name() == ".git" || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || allowed[path] {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(data), directCall) {
			relative, relErr := filepath.Rel(root, path)
			if relErr != nil {
				relative = path
			}
			t.Errorf("%s constructs a command directly; use NewExecCommandContext", relative)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("audit command construction: %v", err)
	}
}
