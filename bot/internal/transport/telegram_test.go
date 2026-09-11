package transport

import "testing"

func TestLocalFilePathUsesFileURI(t *testing.T) {
	path := localFilePath("/downloads/video.mp4")
	if path.NeedsUpload() {
		t.Fatal("local file path must not be sent as multipart upload")
	}
	if got, want := path.SendData(), "file:///downloads/video.mp4"; got != want {
		t.Fatalf("local file URI = %q, want %q", got, want)
	}
}
