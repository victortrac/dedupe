package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMainFunction(t *testing.T) {
	// Create temporary directories for testing
	tempDir1, err := os.MkdirTemp("", "test-dir1")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir1)

	tempDir2, err := os.MkdirTemp("", "test-dir2")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir2)

	// Create test files
	testFiles := map[string]string{
		filepath.Join(tempDir1, "file1.txt"): "content1",
		filepath.Join(tempDir1, "file2.txt"): "content2",
		filepath.Join(tempDir2, "file3.txt"): "content1", // Duplicate of file1.txt
		filepath.Join(tempDir2, "file4.txt"): "content3",
	}

	for path, content := range testFiles {
		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Redirect stdout to capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call main function
	os.Args = []string{"cmd", tempDir1, tempDir2}
	main()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	out, _ := io.ReadAll(r)
	output := string(out)

	// Check if the output contains the expected duplicate files
	expectedOutput := filepath.Join(tempDir1, "file1.txt") + ", " + filepath.Join(tempDir2, "file3.txt")
	if !strings.Contains(output, expectedOutput) {
		t.Errorf("Expected output to contain %s, but got: %s", expectedOutput, output)
	}
}

func TestGetFileHash(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test-file")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	content := "test content"
	if _, err := tempFile.Write([]byte(content)); err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}
	tempFile.Close()

	hash, err := getFileHash(tempFile.Name())
	if err != nil {
		t.Fatalf("getFileHash failed: %v", err)
	}

	expectedHash := "26877eb147d6f408"
	if hash != expectedHash {
		t.Errorf("Expected hash %s, but got %s", expectedHash, hash)
	}
}

func TestAreFilesEqual(t *testing.T) {
	tempFile1, err := os.CreateTemp("", "test-file1")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tempFile1.Name())

	tempFile2, err := os.CreateTemp("", "test-file2")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tempFile2.Name())

	content := "test content"
	if _, err := tempFile1.Write([]byte(content)); err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}
	if _, err := tempFile2.Write([]byte(content)); err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}
	tempFile1.Close()
	tempFile2.Close()

	equal, err := areFilesEqual(tempFile1.Name(), tempFile2.Name())
	if err != nil {
		t.Fatalf("areFilesEqual failed: %v", err)
	}
	if !equal {
		t.Errorf("Expected files to be equal, but they were not")
	}

	// Test with different content
	if err := os.WriteFile(tempFile2.Name(), []byte("different content"), 0644); err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}

	equal, err = areFilesEqual(tempFile1.Name(), tempFile2.Name())
	if err != nil {
		t.Fatalf("areFilesEqual failed: %v", err)
	}
	if equal {
		t.Errorf("Expected files to be different, but they were equal")
	}
}
