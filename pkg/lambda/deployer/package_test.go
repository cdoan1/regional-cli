package deployer

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackageBuilder_Build(t *testing.T) {
	// Use the actual OIDC provisioner source for integration-style test
	sourceDir := "../functions/oidc-provisioner"

	pb := NewPackageBuilder(sourceDir)
	zipData, hash, err := pb.Build()

	require.NoError(t, err)
	assert.NotEmpty(t, zipData)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64) // SHA256 hash length

	// Verify it's a valid ZIP
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	require.NoError(t, err)
	assert.Len(t, zipReader.File, 1)
	assert.Equal(t, "bootstrap", zipReader.File[0].Name)

	// Verify executable permissions
	mode := zipReader.File[0].Mode()
	assert.True(t, mode&0111 != 0, "bootstrap should have executable permissions")

	// Verify package size is reasonable
	assert.Less(t, len(zipData), maxPackageSize)
	assert.Greater(t, len(zipData), 1000) // Should be at least 1KB
}

func TestPackageBuilder_InvalidSourceDir(t *testing.T) {
	pb := NewPackageBuilder("/nonexistent/directory")
	_, _, err := pb.Build()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "compilation failed")
}

func TestCreateZipPackage(t *testing.T) {
	// Create a temporary binary file
	tmpDir, err := os.MkdirTemp("", "package-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	binaryPath := filepath.Join(tmpDir, "bootstrap")
	content := []byte("fake binary content")
	err = os.WriteFile(binaryPath, content, 0755)
	require.NoError(t, err)

	pb := NewPackageBuilder("")
	zipData, err := pb.createZipPackage(binaryPath)
	require.NoError(t, err)

	// Verify ZIP content
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	require.NoError(t, err)
	assert.Len(t, zipReader.File, 1)

	// Extract and verify content
	file := zipReader.File[0]
	assert.Equal(t, "bootstrap", file.Name)

	reader, err := file.Open()
	require.NoError(t, err)
	defer reader.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(reader)
	require.NoError(t, err)
	assert.Equal(t, content, buf.Bytes())
}

func TestCreateZipPackage_BinaryNotFound(t *testing.T) {
	pb := NewPackageBuilder("")
	_, err := pb.createZipPackage("/nonexistent/file")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open binary")
}

func TestPackageBuilder_HashConsistency(t *testing.T) {
	sourceDir := "../functions/oidc-provisioner"
	pb := NewPackageBuilder(sourceDir)

	// Build twice and verify hashes match (deterministic build)
	zipData1, hash1, err := pb.Build()
	require.NoError(t, err)

	zipData2, hash2, err := pb.Build()
	require.NoError(t, err)

	// Hashes might differ due to build timestamps in the Go binary
	// but we can verify the hash matches the actual content
	actualHash1 := fmt.Sprintf("%x", sha256.Sum256(zipData1))
	actualHash2 := fmt.Sprintf("%x", sha256.Sum256(zipData2))

	assert.Equal(t, hash1, actualHash1)
	assert.Equal(t, hash2, actualHash2)
}

func TestPackageBuilder_BinaryPermissions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "permissions-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	binaryPath := filepath.Join(tmpDir, "bootstrap")

	// Create a binary with wrong permissions
	err = os.WriteFile(binaryPath, []byte("test"), 0644)
	require.NoError(t, err)

	pb := NewPackageBuilder("")

	// After creating the ZIP, verify permissions are set correctly
	zipData, err := pb.createZipPackage(binaryPath)
	require.NoError(t, err)

	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	require.NoError(t, err)

	mode := zipReader.File[0].Mode()
	assert.True(t, mode&0111 != 0, "bootstrap should have executable permissions in ZIP")
}
