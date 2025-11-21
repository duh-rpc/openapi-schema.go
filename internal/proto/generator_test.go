package proto_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/require"
)

func TestBufGenerate(t *testing.T) {
	if _, err := exec.LookPath("buf"); err != nil {
		t.Skip("buf not found in PATH, skipping integration test")
	}
	if _, err := exec.LookPath("protoc-gen-go"); err != nil {
		t.Skip("protoc-gen-go not found in PATH, skipping integration test")
	}

	const openapi = `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        userId:
          type: string
        email:
          type: string
        age:
          type: integer
          format: int32
        isActive:
          type: boolean
        tags:
          type: array
          items:
            type: string
        createdAt:
          type: string
          format: date-time
    Status:
      type: string
      enum:
        - active
        - inactive
        - pending
`

	result, err := schema.Convert([]byte(openapi), schema.ConvertOptions{
		PackageName: "testapi",
		PackagePath: "github.com/example/proto/v1/testapi",
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	dir := t.TempDir()
	protoFile := filepath.Join(dir, "test.proto")
	err = os.WriteFile(protoFile, result.Protobuf, 0644)
	require.NoError(t, err)

	bufYAML := `version: v2
modules:
  - path: .
lint:
  use:
    - STANDARD
breaking:
  use:
    - FILE
`
	bufYAMLFile := filepath.Join(dir, "buf.yaml")
	err = os.WriteFile(bufYAMLFile, []byte(bufYAML), 0644)
	require.NoError(t, err)

	bufGenYAML := `version: v2
managed:
  enabled: true
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen
    opt:
      - paths=source_relative
`
	bufGenYAMLFile := filepath.Join(dir, "buf.gen.yaml")
	err = os.WriteFile(bufGenYAMLFile, []byte(bufGenYAML), 0644)
	require.NoError(t, err)

	cmd := exec.Command("buf", "generate")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "buf generate failed: %s", string(output))

	genFile := filepath.Join(dir, "gen", "test.pb.go")
	_, err = os.Stat(genFile)
	require.NoError(t, err, "expected generated Go file at %s", genFile)
}
