package api

import (
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

var mediaContentTypes = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".heic": "image/heic",
	".pdf":  "application/pdf",
}

// mediaHandler streams a media file from the vault's media directory.
// Auth is the standard header middleware; the path parameter must land
// inside the media dir — traversal or non-media paths are rejected.
func (s *Server) mediaHandler(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	if rel == "" {
		WriteError(w, "missing required parameter: path", "MEDIA_MISSING_PATH", http.StatusBadRequest)
		return
	}

	mediaRoot := s.vault.MediaPath()
	// Normalize to inbox-relative so both conventions resolve:
	//   "media/x.jpg" (media-relative) and "khayal/media/x.jpg"
	// (vault-relative, the shape stored in notes' source_file). Join
	// against the inbox, then require the final path to live inside
	// the media dir.
	inboxRoot := filepath.Dir(mediaRoot)
	rel = strings.TrimPrefix(rel, s.config.Vault.InboxDir+"/")
	clean := path.Clean("/" + rel)
	full := filepath.Join(inboxRoot, clean)

	if !strings.HasPrefix(full, mediaRoot+string(filepath.Separator)) {
		s.logger.Warn("media path rejected", "path", rel)
		WriteError(w, "invalid media path", "MEDIA_INVALID_PATH", http.StatusBadRequest)
		return
	}

	ext := strings.ToLower(filepath.Ext(full))
	ct, ok := mediaContentTypes[ext]
	if !ok {
		WriteError(w, "unsupported media type", "MEDIA_UNSUPPORTED_TYPE", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", ct)
	// Private: no intermediary may cache token-fetched media.
	w.Header().Set("Cache-Control", "private, max-age=3600")
	http.ServeFile(w, r, full)
}
