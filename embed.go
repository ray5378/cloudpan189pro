package embed

import (
	"io/fs"
	"os"
)

// External static files mode: serve from ./public if present
func StaticFS() (fs.FS, bool) {
	path := "./public"
	if st, err := os.Stat(path); err == nil && st.IsDir() {
		return os.DirFS(path), true
	}
	return nil, false
}
