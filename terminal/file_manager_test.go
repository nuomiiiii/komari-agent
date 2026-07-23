package terminal

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type recordingFileWriter struct {
	mu       sync.Mutex
	messages []map[string]any
}

func (writer *recordingFileWriter) writeJSON(value any) error {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	message, ok := value.(map[string]any)
	if ok {
		writer.messages = append(writer.messages, message)
	}
	return nil
}

func (writer *recordingFileWriter) last(t *testing.T) map[string]any {
	t.Helper()
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if len(writer.messages) == 0 {
		t.Fatal("expected a file manager response")
	}
	return writer.messages[len(writer.messages)-1]
}

func responseOK(t *testing.T, response map[string]any) bool {
	t.Helper()
	ok, exists := response["ok"].(bool)
	if !exists {
		t.Fatalf("response has no boolean ok field: %#v", response)
	}
	return ok
}

func uploadIDFromResponse(t *testing.T, response map[string]any) string {
	t.Helper()
	data, ok := response["data"].(map[string]any)
	if !ok {
		t.Fatalf("response has no data: %#v", response)
	}
	id, ok := data["upload_id"].(string)
	if !ok || id == "" {
		t.Fatalf("response has no upload id: %#v", response)
	}
	return id
}

func TestSQLiteProtectionRecognizesNameAndHeader(t *testing.T) {
	directory := t.TempDir()
	byName := filepath.Join(directory, "metrics.sqlite3")
	if err := os.WriteFile(byName, []byte("not a database"), 0o600); err != nil {
		t.Fatal(err)
	}
	byHeader := filepath.Join(directory, "metrics.data")
	payload := append([]byte("SQLite format 3\x00"), make([]byte, 32)...)
	if err := os.WriteFile(byHeader, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{byName, byHeader} {
		if err := ensureFileAllowed(path); err == nil {
			t.Fatalf("expected SQLite protection for %s", path)
		}
	}
}

func TestDirectoryWithSQLiteCannotBeRemovedOrRenamed(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "protected")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	database := filepath.Join(directory, "komari.db")
	if err := os.WriteFile(database, []byte("database"), 0o600); err != nil {
		t.Fatal(err)
	}
	writer := &recordingFileWriter{}
	manager := newFileManager(writer)
	defer manager.close()

	manager.remove(fileRequest{Type: "file.delete", ID: "delete", Path: directory, Recursive: true})
	if responseOK(t, writer.last(t)) {
		t.Fatal("directory containing SQLite was deleted")
	}
	if _, err := os.Stat(database); err != nil {
		t.Fatalf("protected database changed after delete attempt: %v", err)
	}

	destination := filepath.Join(filepath.Dir(directory), "renamed")
	manager.rename(fileRequest{Type: "file.rename", ID: "rename", Path: directory, Destination: destination})
	if responseOK(t, writer.last(t)) {
		t.Fatal("directory containing SQLite was renamed")
	}
	if _, err := os.Stat(database); err != nil {
		t.Fatalf("protected database changed after rename attempt: %v", err)
	}
}

func TestFilesystemRootCannotBeRenamed(t *testing.T) {
	writer := &recordingFileWriter{}
	manager := newFileManager(writer)
	defer manager.close()

	manager.rename(fileRequest{
		Type:        "file.rename",
		ID:          "rename-root",
		Path:        filesystemRoots()[0],
		Destination: filepath.Join(t.TempDir(), "renamed-root"),
	})
	if responseOK(t, writer.last(t)) {
		t.Fatal("filesystem root was renamed")
	}
}

func TestUploadRejectsDisguisedSQLiteContent(t *testing.T) {
	target := filepath.Join(t.TempDir(), "innocent.txt")
	payload := append([]byte("SQLite format 3\x00"), make([]byte, 64)...)
	writer := &recordingFileWriter{}
	manager := newFileManager(writer)
	defer manager.close()

	manager.startUpload(fileRequest{Type: "file.upload.start", ID: "start", Path: target, Size: int64(len(payload))})
	uploadID := uploadIDFromResponse(t, writer.last(t))
	manager.uploadChunk(fileRequest{Type: "file.upload.chunk", ID: "chunk", UploadID: uploadID, Data: base64.StdEncoding.EncodeToString(payload)})
	if !responseOK(t, writer.last(t)) {
		t.Fatalf("upload chunk failed: %#v", writer.last(t))
	}
	manager.finishUpload(fileRequest{Type: "file.upload.finish", ID: "finish", UploadID: uploadID})
	if responseOK(t, writer.last(t)) {
		t.Fatal("disguised SQLite upload unexpectedly succeeded")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("disguised SQLite upload left a target file: %v", err)
	}
}

func TestRegularUploadCompletes(t *testing.T) {
	target := filepath.Join(t.TempDir(), "notes.txt")
	payload := []byte("regular remote file\n")
	writer := &recordingFileWriter{}
	manager := newFileManager(writer)
	defer manager.close()

	manager.startUpload(fileRequest{Type: "file.upload.start", ID: "start", Path: target, Size: int64(len(payload))})
	uploadID := uploadIDFromResponse(t, writer.last(t))
	manager.uploadChunk(fileRequest{Type: "file.upload.chunk", ID: "chunk", UploadID: uploadID, Data: base64.StdEncoding.EncodeToString(payload)})
	manager.finishUpload(fileRequest{Type: "file.upload.finish", ID: "finish", UploadID: uploadID})
	if !responseOK(t, writer.last(t)) {
		t.Fatalf("regular upload failed: %#v", writer.last(t))
	}
	actual, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != string(payload) {
		t.Fatalf("uploaded content = %q, want %q", actual, payload)
	}
}

func TestOverwriteUploadReplacesRegularFile(t *testing.T) {
	target := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(target, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	payload := []byte("replacement")
	writer := &recordingFileWriter{}
	manager := newFileManager(writer)
	defer manager.close()

	manager.startUpload(fileRequest{Type: "file.upload.start", ID: "start", Path: target, Size: int64(len(payload)), Overwrite: true})
	uploadID := uploadIDFromResponse(t, writer.last(t))
	manager.uploadChunk(fileRequest{Type: "file.upload.chunk", ID: "chunk", UploadID: uploadID, Data: base64.StdEncoding.EncodeToString(payload)})
	manager.finishUpload(fileRequest{Type: "file.upload.finish", ID: "finish", UploadID: uploadID})
	if !responseOK(t, writer.last(t)) {
		t.Fatalf("overwrite upload failed: %#v", writer.last(t))
	}
	actual, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != string(payload) {
		t.Fatalf("overwritten content = %q, want %q", actual, payload)
	}
	backups, err := filepath.Glob(filepath.Join(filepath.Dir(target), ".komari-backup-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 0 {
		t.Fatalf("overwrite left backup files: %v", backups)
	}
}

func TestDownloadErrorUsesDownloadErrorFrame(t *testing.T) {
	writer := &recordingFileWriter{}
	manager := newFileManager(writer)
	defer manager.close()
	manager.download(fileRequest{Type: "file.download", ID: "download", Path: filepath.Join(t.TempDir(), "missing.txt")})
	response := writer.last(t)
	if response["type"] != "file.download.error" || response["id"] != "download" {
		t.Fatalf("unexpected download error response: %#v", response)
	}
}
