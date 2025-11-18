package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuilder_Build(t *testing.T) {
	// Get absolute path to examples directory
	examplesDir, err := filepath.Abs("../../../examples/simple-api")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	tests := []struct {
		name    string
		dir     string
		wantErr bool
	}{
		{
			name:    "valid go project",
			dir:     examplesDir,
			wantErr: false,
		},
		{
			name:    "invalid directory",
			dir:     "/nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder(tt.dir)
			outputPath, err := builder.Build()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Build() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Build() unexpected error: %v", err)
				return
			}

			// Check that output file exists
			if _, statErr := os.Stat(outputPath); statErr != nil {
				t.Errorf("Build() output file does not exist: %s", outputPath)
			}

			// Clean up
			_ = os.Remove(outputPath)
		})
	}
}

func TestBuilder_GetOutputPath(t *testing.T) {
	dir := "/path/to/project"
	builder := NewBuilder(dir)

	outputPath := builder.GetOutputPath()

	// Should be in .transire directory
	if !filepath.IsAbs(outputPath) {
		t.Errorf("GetOutputPath() should return absolute path, got: %s", outputPath)
	}

	if filepath.Base(filepath.Dir(outputPath)) != ".transire" {
		t.Errorf("GetOutputPath() should be in .transire directory, got: %s", outputPath)
	}
}

func TestBuilder_Clean(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create .transire directory
	transireDir := filepath.Join(tmpDir, ".transire")
	if err := os.MkdirAll(transireDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create a dummy binary
	binaryPath := filepath.Join(transireDir, "app")
	if err := os.WriteFile(binaryPath, []byte("dummy"), 0755); err != nil {
		t.Fatalf("Failed to create dummy binary: %v", err)
	}

	builder := NewBuilder(tmpDir)

	// Clean should remove the .transire directory
	if err := builder.Clean(); err != nil {
		t.Errorf("Clean() unexpected error: %v", err)
	}

	// Verify .transire directory is removed
	if _, err := os.Stat(transireDir); !os.IsNotExist(err) {
		t.Errorf("Clean() should remove .transire directory")
	}
}
