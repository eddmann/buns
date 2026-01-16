package exec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePackageSpec(t *testing.T) {
	tests := []struct {
		spec        string
		wantName    string
		wantVersion string
	}{
		{"lodash", "lodash", ""},
		{"lodash@^4.0", "lodash", "^4.0"},
		{"lodash@4.17.21", "lodash", "4.17.21"},
		{"@types/node", "@types/node", ""},
		{"@types/node@^20.0", "@types/node", "^20.0"},
		{"@org/package@1.0.0", "@org/package", "1.0.0"},
		{"express@>=4.0.0", "express", ">=4.0.0"},
		{"chalk@~5.0.0", "chalk", "~5.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			name, version := parsePackageSpec(tt.spec)
			if name != tt.wantName {
				t.Errorf("parsePackageSpec(%q) name = %q, want %q", tt.spec, name, tt.wantName)
			}
			if version != tt.wantVersion {
				t.Errorf("parsePackageSpec(%q) version = %q, want %q", tt.spec, version, tt.wantVersion)
			}
		})
	}
}

func TestExecScript_runs_in_callers_working_directory(t *testing.T) {
	workDir := t.TempDir()
	scriptDir := t.TempDir()

	// Create a fake "bun" binary that ignores "run" and executes the script
	fakeBun := filepath.Join(scriptDir, "fakebun")
	fakeBunScript := `#!/bin/sh
# Skip "run" argument, execute the rest
shift
exec /bin/sh "$@"
`
	if err := os.WriteFile(fakeBun, []byte(fakeBunScript), 0755); err != nil {
		t.Fatalf("failed to write fake bun: %v", err)
	}

	// Create a shell script that writes a marker file in the current directory
	scriptPath := filepath.Join(scriptDir, "marker.sh")
	script := `#!/bin/sh
touch marker.txt
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	// Change to work directory before running
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("failed to change to work dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Create a minimal runner and execute the script
	r := &Runner{verbose: false, quiet: true}
	exitCode, err := r.execScript(fakeBun, scriptPath, nil, "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	// Verify marker file was created in workDir (not scriptDir)
	markerInWorkDir := filepath.Join(workDir, "marker.txt")
	markerInScriptDir := filepath.Join(scriptDir, "marker.txt")

	if !fileExists(markerInWorkDir) {
		t.Error("marker.txt not created in working directory")
	}

	if fileExists(markerInScriptDir) {
		t.Error("marker.txt incorrectly created in script directory")
	}
}

func TestExecScript_returns_nonzero_exit_code(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a fake "bun" binary
	fakeBun := filepath.Join(tmpDir, "fakebun")
	fakeBunScript := `#!/bin/sh
shift
exec /bin/sh "$@"
`
	if err := os.WriteFile(fakeBun, []byte(fakeBunScript), 0755); err != nil {
		t.Fatalf("failed to write fake bun: %v", err)
	}

	scriptPath := filepath.Join(tmpDir, "fail.sh")
	script := `#!/bin/sh
exit 42
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	r := &Runner{verbose: false, quiet: true}
	exitCode, err := r.execScript(fakeBun, scriptPath, nil, "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exitCode != 42 {
		t.Errorf("exit code = %d, want 42", exitCode)
	}
}

func TestExecScript_passes_arguments_to_script(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "args.txt")

	// Create a fake "bun" binary
	fakeBun := filepath.Join(tmpDir, "fakebun")
	fakeBunScript := `#!/bin/sh
shift
exec /bin/sh "$@"
`
	if err := os.WriteFile(fakeBun, []byte(fakeBunScript), 0755); err != nil {
		t.Fatalf("failed to write fake bun: %v", err)
	}

	scriptPath := filepath.Join(tmpDir, "argcheck.sh")
	script := `#!/bin/sh
echo "$1" > "` + outputFile + `"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	r := &Runner{verbose: false, quiet: true}
	exitCode, err := r.execScript(fakeBun, scriptPath, []string{"test-value"}, "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if string(content) != "test-value\n" {
		t.Errorf("argument = %q, want %q", string(content), "test-value\n")
	}
}

func TestExecScript_sets_NODE_PATH_when_deps_provided(t *testing.T) {
	tmpDir := t.TempDir()
	depsDir := filepath.Join(tmpDir, "deps")
	outputFile := filepath.Join(tmpDir, "nodepath.txt")

	if err := os.MkdirAll(depsDir, 0755); err != nil {
		t.Fatalf("failed to create deps dir: %v", err)
	}

	// Create a fake "bun" binary
	fakeBun := filepath.Join(tmpDir, "fakebun")
	fakeBunScript := `#!/bin/sh
shift
exec /bin/sh "$@"
`
	if err := os.WriteFile(fakeBun, []byte(fakeBunScript), 0755); err != nil {
		t.Fatalf("failed to write fake bun: %v", err)
	}

	scriptPath := filepath.Join(tmpDir, "nodepath.sh")
	script := `#!/bin/sh
echo "$NODE_PATH" > "` + outputFile + `"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	r := &Runner{verbose: false, quiet: true}
	exitCode, err := r.execScript(fakeBun, scriptPath, nil, depsDir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	expectedNodePath := filepath.Join(depsDir, "node_modules")
	if string(content) != expectedNodePath+"\n" {
		t.Errorf("NODE_PATH = %q, want %q", string(content), expectedNodePath+"\n")
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
