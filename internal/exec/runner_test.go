package exec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddmann/buns/internal/cache"
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

func TestBuildTypeCheckPackages(t *testing.T) {
	packages := []string{"zod@^3.0", "chalk@^5.0"}

	t.Run("pins bun types to resolved bun version", func(t *testing.T) {
		got := buildTypeCheckPackages(packages, "1.3.13", true)
		want := []string{"zod@^3.0", "chalk@^5.0", typeScriptPackage, "@types/bun@1.3.13"}

		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Errorf("packages = %v, want %v", got, want)
		}
	})

	t.Run("falls back to latest bun types", func(t *testing.T) {
		got := buildTypeCheckPackages(packages, "1.3.13", false)
		want := []string{"zod@^3.0", "chalk@^5.0", typeScriptPackage, "@types/bun"}

		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Errorf("packages = %v, want %v", got, want)
		}
	})
}

func TestBuildTypeCheckConfig(t *testing.T) {
	config := buildTypeCheckConfig("/tmp/script.ts", "/tmp/typecheck")
	options := config.CompilerOptions

	if len(config.Files) != 1 || config.Files[0] != "/tmp/script.ts" {
		t.Errorf("files = %v, want script path", config.Files)
	}
	if !options.NoEmit {
		t.Error("noEmit should be enabled")
	}
	if !options.Strict {
		t.Error("strict should be enabled")
	}
	if options.Module != "Preserve" {
		t.Errorf("module = %q, want Preserve", options.Module)
	}
	if options.ModuleResolution != "bundler" {
		t.Errorf("moduleResolution = %q, want bundler", options.ModuleResolution)
	}
	if len(options.Types) != 1 || options.Types[0] != "bun" {
		t.Errorf("types = %v, want [bun]", options.Types)
	}
	if options.BaseURL != "/tmp/typecheck" {
		t.Errorf("baseUrl = %q, want /tmp/typecheck", options.BaseURL)
	}
	if got := options.Paths["*"][0]; got != "node_modules/*" {
		t.Errorf("paths[*][0] = %q, want node_modules/*", got)
	}
}

func TestRunTypeCheckReturnsExitCode(t *testing.T) {
	tmpDir := t.TempDir()
	typeCheckDir := filepath.Join(tmpDir, "typecheck")
	tscPath := filepath.Join(typeCheckDir, "node_modules", "typescript", "lib", "tsc.js")
	if err := os.MkdirAll(filepath.Dir(tscPath), 0755); err != nil {
		t.Fatalf("failed to create fake tsc dir: %v", err)
	}
	if err := os.WriteFile(tscPath, []byte("console.log('fake tsc')\n"), 0644); err != nil {
		t.Fatalf("failed to write fake tsc: %v", err)
	}

	argsFile := filepath.Join(tmpDir, "args.txt")
	fakeBun := filepath.Join(tmpDir, "fakebun")
	script := `#!/bin/sh
printf '%s\n' "$@" > "` + argsFile + `"
exit 2
`
	if err := os.WriteFile(fakeBun, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake bun: %v", err)
	}

	r := &Runner{verbose: false, quiet: true}
	exitCode, err := r.runTypeCheck(fakeBun, typeCheckDir, filepath.Join(tmpDir, "tsconfig.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 2 {
		t.Errorf("exit code = %d, want 2", exitCode)
	}

	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("failed to read fake bun args: %v", err)
	}
	if !strings.Contains(string(args), "node_modules/typescript/lib/tsc.js") {
		t.Errorf("args = %q, want tsc.js path", string(args))
	}
}

func TestTypeCheckScriptInstallsTypecheckDepsAndRunsTSC(t *testing.T) {
	tmpDir := t.TempDir()
	argsFile := filepath.Join(tmpDir, "tsc-args.txt")
	t.Setenv("TSC_ARGS_FILE", argsFile)
	t.Setenv("TSC_EXIT_CODE", "2")

	fakeBun := filepath.Join(tmpDir, "fakebun")
	fakeBunScript := `#!/bin/sh
if [ "$1" = "install" ]; then
	mkdir -p node_modules/typescript/lib node_modules/installed
	printf 'fake tsc\n' > node_modules/typescript/lib/tsc.js
	exit 0
fi
case "$1" in
	*/node_modules/typescript/lib/tsc.js)
		printf '%s\n' "$@" > "$TSC_ARGS_FILE"
		exit "$TSC_EXIT_CODE"
		;;
esac
exit 99
`
	if err := os.WriteFile(fakeBun, []byte(fakeBunScript), 0755); err != nil {
		t.Fatalf("failed to write fake bun: %v", err)
	}

	scriptPath := filepath.Join(tmpDir, "script.ts")
	if err := os.WriteFile(scriptPath, []byte("console.log(Bun.version)\n"), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	c := cache.New(filepath.Join(tmpDir, "cache"))
	r := &Runner{cache: c, verbose: false, quiet: true}
	packages := []string{"zod@^3.0"}
	exitCode, err := r.typeCheckScript(fakeBun, scriptPath, packages, "1.3.13")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 2 {
		t.Errorf("exit code = %d, want 2", exitCode)
	}

	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("failed to read tsc args: %v", err)
	}
	argsText := string(args)
	for _, expectedArg := range []string{"--project", "--pretty", "--noErrorTruncation"} {
		if !strings.Contains(argsText, expectedArg) {
			t.Errorf("tsc args = %q, want %s", argsText, expectedArg)
		}
	}

	typeCheckPackages := buildTypeCheckPackages(packages, "1.3.13", true)
	typeCheckDir := c.TypecheckDirForHash(cache.HashPackages(typeCheckPackages))
	packageJSON, err := os.ReadFile(filepath.Join(typeCheckDir, "package.json"))
	if err != nil {
		t.Fatalf("failed to read typecheck package.json: %v", err)
	}
	packageJSONText := string(packageJSON)
	for _, expected := range []string{"zod", "typescript", "@types/bun"} {
		if !strings.Contains(packageJSONText, expected) {
			t.Errorf("package.json missing %q: %s", expected, packageJSONText)
		}
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
