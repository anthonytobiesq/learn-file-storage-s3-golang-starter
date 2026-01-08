package main

/*
Delete the handler function itself - Find the function that handles GET /api/thumbnails/{videoID} and remove it
Delete the route registration - Find where you registered that GET route in your main.go (or wherever your routes are set up) and remove that line too
Delete the global thumbnail map - Remove the videoThumbnails map declaration (probably near the top of one of your files)
Delete the thumbnail struct - If you have a thumbnail struct definition that was used for the map, remove that too






import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerThumbnailGet(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid video ID", err)
		return
	}

	tn, ok := videoThumbnails[videoID]
	if !ok {
		respondWithError(w, http.StatusNotFound, "Thumbnail not found", nil)
		return
	}

	w.Header().Set("Content-Type", tn.mediaType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(tn.data)))

	_, err = w.Write(tn.data)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error writing response", err)
		return
	}
}
*/
