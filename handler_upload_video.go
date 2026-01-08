package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/media"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
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
		respondWithError(w, http.StatusUnauthorized, "Video belongs to another user, you do not have permission to upload this video", err)
		return
	}

	//limit the size of the request to 1GB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)
	//parse the multipart form, 1 << 30 specifying a 1GB limit in memory, not upload limit
	if err = r.ParseMultipartForm(1 << 30); err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			respondWithError(w, http.StatusBadRequest, "Video is too large", err)
			return
		}
		respondWithError(w, http.StatusBadRequest, "Couldn't parse multipart form", err)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error retrieving file", err)
		return
	}
	defer file.Close()

	mediaType := header.Header.Get("Content-Type")

	videoMediaType, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid media type", err)
		return
	}
	if videoMediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid media type", nil)
		return
	}

	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create temporary file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't write temporary file", err)
		return
	}

	// 1. Move back to the start of the file
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't seek to start of file", err)
		return
	}

	ratio, err := media.GetVideoAspectRatio(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video aspect ratio", err)
		return
	}
	log.Printf("video aspect ratio: %s", ratio)
	
	prefix := ""
	if ratio == "9:16" {
		prefix = "portrait/"
	} else if ratio == "16:9" {
		prefix = "landscape/"
	} else {
		prefix = "other/"
	}

	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't generate random key for video", err)
		return
	}
	//encoding/hex.EncodeToString(key)
	videoKey := hex.EncodeToString(key) + ".mp4"
	videoKey = prefix + videoKey

	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      aws.String(cfg.s3Bucket),
		Key:         aws.String(videoKey),
		Body:        tempFile,
		ContentType: aws.String(videoMediaType),
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't upload video to S3", err)
		return
	}

	//Update the VideoURL of the video record in the database with the S3 bucket and key.
	//S3 URLs are in the format https://<bucket-name>.s3.<region>.amazonaws.com/<key>.
	//Make sure you use the correct region and bucket name!

	videoURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, videoKey)
	video.VideoURL = &videoURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}
	respondWithJSON(w, http.StatusOK, video)

}
