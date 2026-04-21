package casparser

import "testing"

func TestIsCasFile(t *testing.T) {
	if !IsCasFile("movie.mkv.cas") {
		t.Fatalf("expected cas file")
	}
	if IsCasFile("movie.mkv") {
		t.Fatalf("did not expect non-cas file")
	}
}

func TestGetOriginalFileName(t *testing.T) {
	if got := GetOriginalFileName("movie.mkv.cas", nil); got != "movie.mkv" {
		t.Fatalf("unexpected name: %s", got)
	}
	info := &CasInfo{Name: "episode.mp4"}
	if got := GetOriginalFileName("episode.cas", info); got != "episode.mp4" {
		t.Fatalf("unexpected fallback name: %s", got)
	}
}

func TestParseCasContent_JSON(t *testing.T) {
	content := []byte(`{"name":"movie.mkv","size":123,"md5":"abc","sliceMd5":"def"}`)
	info, err := ParseCasContent(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Name != "movie.mkv" || info.Size != 123 || info.MD5 != "ABC" || info.SliceMD5 != "DEF" {
		t.Fatalf("unexpected parsed info: %+v", info)
	}
}

func TestParseCasContent_Base64(t *testing.T) {
	content := []byte("eyJuYW1lIjoiZXBpc29kZS5ta3YiLCJzaXplIjoyMDQ4LCJtZDUiOiJhYmMiLCJzbGljZU1kNSI6ImRlZiJ9")
	info, err := ParseCasContent(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Name != "episode.mkv" || info.Size != 2048 {
		t.Fatalf("unexpected parsed info: %+v", info)
	}
}
