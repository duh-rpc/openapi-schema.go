package schema_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/require"
)

// combinedStyleBSpec declares both forms of the union over the same variant
// properties with matching field numbers: OneofEvent is style B (generates a
// protobuf oneof) and OptionalEvent is the already-supported optional-fields form.
// Wire compatibility means these two encodings are interchangeable on both wires.
const combinedStyleBSpec = `openapi: 3.0.0
info:
  title: Wire Compat
  version: 1.0.0
components:
  schemas:
    OneofEvent:
      type: object
      properties:
        cat_event:
          $ref: '#/components/schemas/Cat'
        dog_event:
          $ref: '#/components/schemas/Dog'
      oneOf:
        - required: [cat_event]
        - required: [dog_event]
    OptionalEvent:
      type: object
      properties:
        cat_event:
          $ref: '#/components/schemas/Cat'
        dog_event:
          $ref: '#/components/schemas/Dog'
    Cat:
      type: object
      properties:
        pet_name:
          type: string
    Dog:
      type: object
      properties:
        pet_name:
          type: string
`

// TestGeneratedStyleBIsWireCompatible is the standing guard for the nested wire-shape
// invariant (Blueprint AC 6). It is the docs/oneof-wire-proof harness re-pointed at
// GENERATED output: it generates the style-B protobuf via Convert, compiles it with
// the real protoc/protoc-gen-go toolchain, and runs the wire-compatibility proof
// (testdata/oneof_wire_proof_test.txt) against that generated code.
//
// This is a LOCAL / opt-in guard: it skips (does not fail) when protoc/protoc-gen-go
// are unavailable. CI does not install the protobuf toolchain, so this test does NOT
// run there — a skip here is not coverage. Run it locally (with protoc + protoc-gen-go
// on PATH) to exercise the real wire round-trip; the always-on ENG-61 coverage is the
// golden tests in convert_oneof_styleb_test.go, which assert the oneof group and
// json_name annotations on every CI run.
func TestGeneratedStyleBIsWireCompatible(t *testing.T) {
	requireTool(t, "protoc")
	requireTool(t, "protoc-gen-go")
	requireTool(t, "go")

	result, err := schema.Convert([]byte(combinedStyleBSpec), schema.ConvertOptions{
		PackageName: "proof",
		PackagePath: "proof/gen;gen",
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Protobuf)
	require.Empty(t, result.Golang)

	proofSrc, err := os.ReadFile(filepath.Join("testdata", "oneof_wire_proof_test.txt"))
	require.NoError(t, err)

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "union.proto"), result.Protobuf, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module proof\n\ngo 1.24\n\nrequire google.golang.org/protobuf v1.36.11\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "proof_test.go"), proofSrc, 0o644))

	runCmd(t, dir, "protoc", "--go_out=.", "--go_opt=module=proof", "union.proto")
	runCmd(t, dir, "go", "mod", "tidy")
	runCmd(t, dir, "go", "test", "./...")
}

func requireTool(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s not available; skipping generated wire-compat regression test", name)
	}
}

func runCmd(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "%s %v failed:\n%s", name, args, out)
}
