package ruledfiles_test

import (
	"fmt"
	"testing"

	"arch-agent/internal/files"
	rf "arch-agent/internal/files/rule"
)

func TestRuledFileSystem_ReadLines(t *testing.T) {
	dir := t.TempDir()
	fs, err := files.NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}

	rfs, err := rf.NewRuledFileSystem(fs)
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name     string
		content  string
		from     *int
		to       *int
		expected string
	}{
		{
			name:     "full file",
			content:  "line1\nline2\nline3\nline4\nline5",
			expected: "line1\nline2\nline3\nline4\nline5",
		},
		{
			name:     "range from 2 to 4",
			content:  "line1\nline2\nline3\nline4\nline5",
			from:     intPtr(2),
			to:       intPtr(4),
			expected: "line2\nline3\nline4",
		},
		{
			name:     "range with nil from",
			content:  "line1\nline2\nline3\nline4\nline5",
			to:       intPtr(3),
			expected: "line1\nline2\nline3",
		},
		{
			name:     "range with nil to",
			content:  "line1\nline2\nline3\nline4\nline5",
			from:     intPtr(4),
			expected: "line4\nline5",
		},
		{
			name:     "from out of bounds",
			content:  "line1\nline2\nline3",
			from:     intPtr(10),
			expected: "line3",
		},
		{
			name:     "to out of bounds",
			content:  "line1\nline2\nline3",
			to:       intPtr(10),
			expected: "line1\nline2\nline3",
		},
		{
			name:     "from > to",
			content:  "line1\nline2\nline3\nline4\nline5",
			from:     intPtr(4),
			to:       intPtr(2),
			expected: "line4",
		},
		{
			name:     "empty file",
			content:  "",
			from:     intPtr(1),
			to:       intPtr(10),
			expected: "",
		},
		{
			name:     "single line",
			content:  "only line",
			expected: "only line",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filePath := "test.txt"
			if err := fs.WriteToFile(filePath, []byte(tc.content)); err != nil {
				t.Fatal(err)
			}

			result, err := rfs.ReadLines(filePath, tc.from, tc.to)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestRuledFileSystem_AppendToFile_CreateNew(t *testing.T) {
	dir := t.TempDir()
	fs, err := files.NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}

	rfs, err := rf.NewRuledFileSystem(fs)
	if err != nil {
		t.Fatal(err)
	}

	filePath := "newfile.txt"
	input := []byte("hello")

	if err := rfs.AppendToFile(filePath, input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := fs.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	if string(result) != string(input) {
		t.Errorf("expected %q, got %q", input, result)
	}
}

func TestRuledFileSystem_AppendToFile_Existing(t *testing.T) {
	dir := t.TempDir()
	fs, err := files.NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}

	rfs, err := rf.NewRuledFileSystem(fs)
	if err != nil {
		t.Fatal(err)
	}

	filePath := "existing.txt"

	if err := fs.WriteToFile(filePath, []byte("hello")); err != nil {
		t.Fatal(err)
	}

	if err := rfs.AppendToFile(filePath, []byte(" world")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := fs.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	expected := "hello world"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestRuledFileSystem_AppendToFile_EmptyInput(t *testing.T) {
	dir := t.TempDir()
	fs, err := files.NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}

	rfs, err := rf.NewRuledFileSystem(fs)
	if err != nil {
		t.Fatal(err)
	}

	filePath := "newfile.txt"

	if err := rfs.AppendToFile(filePath, []byte{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := fs.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty file, got %q", result)
	}
}

func TestRuledFileSystem_AppendToFile_SizeLimit(t *testing.T) {
	dir := t.TempDir()
	fs, err := files.NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}

	rfs, err := rf.NewRuledFileSystem(fs, rf.WithWriteSizeLimit(100))
	if err != nil {
		t.Fatal(err)
	}

	filePath := "test.txt"

	if err := rfs.AppendToFile(filePath, []byte("hello")); err != nil {
		t.Fatalf("first append failed: %v", err)
	}

	if err := rfs.AppendToFile(filePath, []byte(" world")); err != nil {
		t.Fatalf("second append failed: %v", err)
	}

	data, err := fs.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	if len(data) > 100 {
		t.Errorf("file size %d exceeds limit", len(data))
	}
}

func TestRuledFileSystem_ReadLines_SizeLimit(t *testing.T) {
	dir := t.TempDir()
	fs, err := files.NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}

	largeContent := generateLargeContent(100)
	if err := fs.WriteToFile("large.txt", []byte(largeContent)); err != nil {
		t.Fatal(err)
	}

	rfs, err := rf.NewRuledFileSystem(fs, rf.WithReadSizeLimit(50))
	if err != nil {
		t.Fatal(err)
	}

	from := 1
	to := 50

	_, err = rfs.ReadLines("large.txt", &from, &to)
	if err == nil {
		t.Error("expected error for file exceeding size limit")
	}
}

func TestRuledFileSystem_ReadLines_LargeFile(t *testing.T) {
	dir := t.TempDir()
	fs, err := files.NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}

	largeContent := generateLargeContent(1000)
	if err := fs.WriteToFile("large.txt", []byte(largeContent)); err != nil {
		t.Fatal(err)
	}

	rfs, err := rf.NewRuledFileSystem(fs)
	if err != nil {
		t.Fatal(err)
	}

	from := 100
	to := 110

	result, err := rfs.ReadLines("large.txt", &from, &to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !containsLines(result, 10) {
		t.Error("expected 10 lines in result")
	}
}

func TestRuledFileSystem_Delete(t *testing.T) {
	dir := t.TempDir()
	fs, err := files.NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}

	rfs, err := rf.NewRuledFileSystem(fs)
	if err != nil {
		t.Fatal(err)
	}

	filePath := "to-delete.txt"
	if err := fs.WriteToFile(filePath, []byte("content")); err != nil {
		t.Fatal(err)
	}

	if err := rfs.Delete(filePath); err != nil {
		t.Fatal(err)
	}

	if _, err := fs.ReadFile(filePath); err == nil {
		t.Error("expected file to be deleted")
	}
}

func TestRuledFileSystem_WriteFile(t *testing.T) {
	dir := t.TempDir()
	fs, err := files.NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}

	rfs, err := rf.NewRuledFileSystem(fs)
	if err != nil {
		t.Fatal(err)
	}

	filePath := "test.txt"
	data := []byte("test data")

	if err := rfs.WriteFile(filePath, data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := fs.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	if string(result) != string(data) {
		t.Errorf("expected %q, got %q", data, result)
	}
}

func TestRuledFileSystem_ReadDir(t *testing.T) {
	dir := t.TempDir()
	fs, err := files.NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}

	rfs, err := rf.NewRuledFileSystem(fs)
	if err != nil {
		t.Fatal(err)
	}

	if err := fs.WriteToFile("file1.txt", []byte("a")); err != nil {
		t.Fatal(err)
	}
	if err := fs.WriteToFile("file2.txt", []byte("b")); err != nil {
		t.Fatal(err)
	}

	entries, err := rfs.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestRuledFileSystem_PathValidation(t *testing.T) {
	dir := t.TempDir()
	fs, err := files.NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}

	rfs, err := rf.NewRuledFileSystem(fs, rf.WithCleanPathOnly())
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid path",
			path:    "test.txt",
			wantErr: false,
		},
		{
			name:    "path with ..",
			path:    "../test.txt",
			wantErr: true,
		},
		{
			name:    "path with .",
			path:    "./test.txt",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.wantErr {
				if err := fs.WriteToFile(tc.path, []byte("test")); err != nil {
					t.Fatal(err)
				}
			}
			_, err := rfs.ReadFile(tc.path)
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRuledFileSystem_ReadOnlyTextFiles(t *testing.T) {
	dir := t.TempDir()
	fs, err := files.NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}

	rfs, err := rf.NewRuledFileSystem(fs, rf.WithReadOnlyTextFiles())
	if err != nil {
		t.Fatal(err)
	}

	if err := fs.WriteToFile("test.md", []byte("# content")); err != nil {
		t.Fatal(err)
	}

	if _, err := rfs.ReadFile("test.md"); err != nil {
		t.Errorf("should allow text files: %v", err)
	}

	if err := fs.WriteToFile("test.bin", []byte{0, 1, 2}); err != nil {
		t.Fatal(err)
	}

	if _, err := rfs.ReadFile("test.bin"); err == nil {
		t.Error("should reject binary files")
	}
}

func TestRuledFileSystem_WithAccessOnPath(t *testing.T) {
	dir := t.TempDir()
	fs, err := files.NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}

	rfs, err := rf.NewRuledFileSystem(fs, rf.WithAccessOnPath("/protected", true, false, false, false))
	if err != nil {
		t.Fatal(err)
	}

	if err := fs.WriteToFile("/protected/file.txt", []byte("secret")); err != nil {
		t.Fatal(err)
	}

	if _, err := rfs.ReadFile("/protected/file.txt"); err != nil {
		t.Errorf("should allow read on guarded path: %v", err)
	}

	if err := rfs.WriteFile("/protected/file.txt", []byte("modified")); err == nil {
		t.Error("should not allow write on guarded path")
	}

	if err := rfs.AppendToFile("/protected/file.txt", []byte("more")); err == nil {
		t.Error("should not allow append on guarded path")
	}
}

func TestRuledFileSystem_SizeLimit(t *testing.T) {
	dir := t.TempDir()
	fs, err := files.NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}

	rfs, err := rf.NewRuledFileSystem(fs, rf.WithReadSizeLimit(10), rf.WithWriteSizeLimit(10))
	if err != nil {
		t.Fatal(err)
	}

	if err := rfs.WriteFile("small.txt", []byte("12345")); err != nil {
		t.Errorf("small write failed: %v", err)
	}

	if err := rfs.WriteFile("large.txt", []byte("12345678901")); err == nil {
		t.Error("should reject large write")
	}

	if _, err := rfs.ReadFile("small.txt"); err != nil {
		t.Errorf("small read failed: %v", err)
	}
}

func TestRuledFileSystem_WithMount(t *testing.T) {
	dir := t.TempDir()
	fs, err := files.NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}

	rfs, err := rf.NewRuledFileSystem(fs, rf.WithMount("/mnt/shared", "/shared"))
	if err != nil {
		t.Fatal(err)
	}

	if err := fs.WriteToFile("/mnt/shared/file.txt", []byte("content")); err != nil {
		t.Fatal(err)
	}

	if _, err := rfs.ReadFile("/shared/file.txt"); err != nil {
		t.Errorf("mount not working: %v", err)
	}
}

func generateLargeContent(numLines int) string {
	var result string
	for i := 1; i <= numLines; i++ {
		result += fmt.Sprintf("line %d\n", i)
	}
	return result
}

func containsLines(s string, count int) bool {
	lines := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines++
		}
	}
	return lines >= count
}

func intPtr(n int) *int {
	return &n
}
