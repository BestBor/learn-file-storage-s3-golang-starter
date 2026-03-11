package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const (
	maxMemory int64 = 10 << 20
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

	// TODO: implement the upload here
	// Load file from form
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "couldn't parse multipart form", err)
		return
	}

	fileData, headers, err := r.FormFile("thumbnail") // html input name
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer fileData.Close()

	// mediaType := headers.Header.Get("Content-Type") // This returns mimetype e.g: image/png
	mediaType, _, err := mime.ParseMediaType(headers.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to extract mediaType", err)
		return
	}

	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "file type invalid", err)
		return
	}

	// V1: thumbnail in memory

	// imgData, err := io.ReadAll(fileData)
	// if err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "couldn't read image data", err)
	// 	return
	// }

	// imgDataStr := base64.StdEncoding.EncodeToString(imgData)
	// dataUrl := fmt.Sprintf("data:%s;base64,%s", mediaType, imgDataStr)

	// videoThumbnails[video.ID] = thumbnail{
	// 	data:      imgData,
	// 	mediaType: mediaType,
	// }

	// newURL := fmt.Sprintf("http://localhost:%v/api/thumbnails/%s", cfg.port, videoIDString)

	// -----------------------------------------------------------------------------------------
	// V2: save file using videoID as name

	// ext := filepath.Ext(headers.Filename)

	// fileName := fmt.Sprintf("%s%s", videoIDString, ext)

	// savePath := filepath.Join(cfg.assetsRoot, fileName)
	// publicPath := "/assets/" + fileName

	// file, err := os.Create(savePath)
	// if err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "failed to create new file", err)
	// 	return
	// }
	// defer file.Close()

	// _, err = io.Copy(file, fileData)
	// if err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "failed to write file", err)
	// 	return
	// }

	// newURL := fmt.Sprintf("http://localhost:%v%s", cfg.port, publicPath)

	// ----------------------------------------------------------------------------------
	// V3: Generate random name to avoid cache

	ext := filepath.Ext(headers.Filename)
	// 32 random bytes
	b := make([]byte, 32)
	_, err = rand.Read(b)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to generate random id", err)
		return
	}
	randName := base64.RawURLEncoding.EncodeToString(b)
	fileName := randName + ext

	// route
	savePath := filepath.Join(cfg.assetsRoot, fileName)
	publicPath := "/assets/" + fileName

	// create file
	file, err := os.Create(savePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to create new file", err)
		return
	}
	defer file.Close()

	// copy data
	_, err = io.Copy(file, fileData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to write file", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to retrieve video", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", errors.New("request made by unauthorized user"))
		return
	}
	// Old Way:
	// videoThumbnails[video.ID] = thumbnail{
	// 	data:      imgData,
	// 	mediaType: mediaType,
	// }
	// newURL := fmt.Sprintf("http://localhost:%v/api/thumbnails/%s", cfg.port, videoIDString)
	// newURL := fmt.Sprintf("http://localhost:%v%s", cfg.port, publicPath)

	newURL := fmt.Sprintf("http://localhost:%v%s", cfg.port, publicPath)
	video.ThumbnailURL = &newURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "internal server error", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
