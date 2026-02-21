package deployer

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	maxPackageSize = 50 * 1024 * 1024 // 50MB limit for Lambda packages
)

// PackageBuilder builds Lambda deployment packages
type PackageBuilder struct {
	sourceDir string
}

// NewPackageBuilder creates a new package builder
func NewPackageBuilder(sourceDir string) *PackageBuilder {
	return &PackageBuilder{
		sourceDir: sourceDir,
	}
}

// Build compiles the Go binary and packages it into a ZIP file
func (pb *PackageBuilder) Build() ([]byte, string, error) {
	// Create temporary directory for build
	tmpDir, err := os.MkdirTemp("", "lambda-build-*")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Cross-compile for Linux/AMD64
	binaryPath := filepath.Join(tmpDir, "bootstrap")
	if err := pb.compileBinary(binaryPath); err != nil {
		return nil, "", fmt.Errorf("failed to compile binary: %w", err)
	}

	// Create ZIP package
	zipData, err := pb.createZipPackage(binaryPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create zip package: %w", err)
	}

	// Validate package size
	if len(zipData) > maxPackageSize {
		return nil, "", fmt.Errorf("package size %d bytes exceeds maximum %d bytes", len(zipData), maxPackageSize)
	}

	// Calculate SHA256 hash
	hash := sha256.Sum256(zipData)
	hashStr := fmt.Sprintf("%x", hash)

	return zipData, hashStr, nil
}

// compileBinary cross-compiles the Go binary for Linux/AMD64
func (pb *PackageBuilder) compileBinary(outputPath string) error {
	cmd := exec.Command("go", "build", "-ldflags", "-s -w", "-o", outputPath, pb.sourceDir)
	cmd.Env = append(os.Environ(),
		"GOOS=linux",
		"GOARCH=amd64",
		"CGO_ENABLED=0",
		"GOTOOLCHAIN=auto",
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compilation failed: %w, stderr: %s", err, stderr.String())
	}

	// Verify binary was created
	if _, err := os.Stat(outputPath); err != nil {
		return fmt.Errorf("binary not found after compilation: %w", err)
	}

	// Set executable permissions (important for Lambda)
	if err := os.Chmod(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to set binary permissions: %w", err)
	}

	return nil
}

// createZipPackage creates a ZIP archive containing the binary
func (pb *PackageBuilder) createZipPackage(binaryPath string) ([]byte, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Open the binary file
	file, err := os.Open(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open binary: %w", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat binary: %w", err)
	}

	// Create ZIP file header with executable permissions
	header, err := zip.FileInfoHeader(fileInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to create zip header: %w", err)
	}

	// Set name to "bootstrap" (required for custom runtime)
	header.Name = "bootstrap"
	header.Method = zip.Deflate

	// Preserve executable permissions in ZIP
	header.SetMode(0755)

	// Create file entry in ZIP
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return nil, fmt.Errorf("failed to create zip entry: %w", err)
	}

	// Copy binary content
	if _, err := io.Copy(writer, file); err != nil {
		return nil, fmt.Errorf("failed to write binary to zip: %w", err)
	}

	// Close ZIP writer
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close zip writer: %w", err)
	}

	return buf.Bytes(), nil
}
