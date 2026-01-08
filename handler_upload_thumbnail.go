package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't get file", err)
		return
	}
	defer file.Close()

	mediaType := header.Header.Get("Content-Type")
	//If the media type isn't either image/jpeg or image/png, respond with an error (respondWithError helper)
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Invalid media type", nil)
		return
	}

	//data, err := io.ReadAll(file)
	//if err != nil {
	//respondWithError(w, http.StatusInternalServerError, "Couldn't read file", err)
	//return
	//}

	key := make([]byte, 32)
	_, err = rand.Read(key) // We only care about the error here
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't generate random key for thumbnail", err)
		return
	}

	thumbnailRandomBaseString := base64.RawURLEncoding.EncodeToString(key) // Encode the entire key slice

	//encoding/base64
	//refactor to store files on disk
	//encodedData := base64.StdEncoding.EncodeToString(data)

	fileExtension := strings.Split(mediaType, "/")[1]

	videoFileExtension := fmt.Sprintf("%s.%s", thumbnailRandomBaseString, fileExtension)

	// use videoID to create a unique filepath using filepath.Join and cfg.assetsRoot
	assetFilepath := filepath.Join(cfg.assetsRoot, videoFileExtension)

	thumbnailFile, err := os.Create(assetFilepath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create thumbnail file", err)
		return
	}
	defer thumbnailFile.Close()

	_, err = io.Copy(thumbnailFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't write thumbnail file", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, http.StatusNotFound, "Video not found in database", err)
			return
		}
		respondWithError(w, http.StatusNotFound, "Couldn't get video", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Video belongs to another user, you do not have permission to post a thumbnail", err)
		return
	}

	//http://localhost:<port>/assets/<videoID>.<file_extension>
	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, videoFileExtension)

	video.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)

	/*
		newThumbnail := thumbnail{
			data:      encodedData,
			mediaType: mediaType,
		}

		videoThumbnails[videoID] = newThumbnail
		//http://localhost:<port>/api/thumbnails/{videoID}
		thumbnailURL := fmt.Sprintf("http://localhost:%s/api/thumbnails/%s", cfg.port, videoID)
		video.ThumbnailURL = &thumbnailURL

		err = cfg.db.UpdateVideo(video)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
			return
		}

		respondWithJSON(w, http.StatusOK, video)

	*/
}
