package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/rohits-web03/cryptodrop/internal/models"
	"github.com/rohits-web03/cryptodrop/internal/repositories"
	"github.com/rohits-web03/cryptodrop/internal/utils"
)

// GET /api/v1/share/{token}
// GetSharedFiles godoc
// @Summary Retrieve shared file details
// @Description Returns metadata (name, size, contentType, index) of all files in a shared transfer.
// @Tags Share
// @Accept json
// @Produce json
// @Param token path string true "Share token"
// @Success 200 {object} utils.Payload "Files retrieved successfully"
// @Failure 400 {object} utils.Payload "Missing or invalid token"
// @Failure 404 {object} utils.Payload "Invalid or expired share link"
// @Failure 410 {object} utils.Payload "Share link has expired"
// @Router /api/v1/share/{token} [get]
func GetSharedFiles(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		utils.JSONResponse(w, http.StatusBadRequest, utils.Payload{
			Success: false,
			Message: "Missing share token",
		})
		return
	}

	db := repositories.DB
	var transfer models.Transfer

	// Fetch transfer and preload its files
	err := db.Preload("Files").
		Where("token = ? AND deleted = ?", token, false).
		First(&transfer).Error

	if err != nil {
		utils.JSONResponse(w, http.StatusNotFound, utils.Payload{
			Success: false,
			Message: "Invalid or expired share link",
		})
		return
	}

	// Check expiry
	// Ensure the transfer link is still valid
	if time.Now().After(transfer.ExpiresAt) {
		utils.JSONResponse(w, http.StatusGone, utils.Payload{
			Success: false,
			Message: "This link has expired",
		})
		return
	}

	// Prepare safe response
	files := make([]map[string]interface{}, 0, len(transfer.Files))
	for _, f := range transfer.Files {
		files = append(files, map[string]interface{}{
			"name":        f.Filename,
			"size":        f.Size,
			"contentType": f.ContentType,
			"index":       f.Index,
		})
	}

	utils.JSONResponse(w, http.StatusOK, utils.Payload{
		Success: true,
		Message: "Files retrieved successfully",
		Data: map[string]any{
			"expires_at": transfer.ExpiresAt,
			"files":      files,
		},
	})
}

// GET /api/v1/share/{token}/presign-download/{index}
// PresignDownload godoc
// @Summary Generate a presigned download URL
// @Description Returns a temporary signed URL to download a specific file (by index) from a shared transfer.
// @Tags Share
// @Accept json
// @Produce json
// @Param token path string true "Share token"
// @Param index path int true "File index"
// @Success 200 {object} utils.Payload "Presigned download URL generated successfully"
// @Failure 400 {object} utils.Payload "Missing or invalid parameters"
// @Failure 404 {object} utils.Payload "File not found or invalid share link"
// @Failure 410 {object} utils.Payload "Share link has expired"
// @Router /api/v1/share/{token}/presign-download/{index} [get]
func PresignDownload(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	indexStr := r.PathValue("index")
	if token == "" || indexStr == "" {
		utils.JSONResponse(w, http.StatusBadRequest, utils.Payload{
			Success: false,
			Message: "Missing token or index",
		})
		return
	}

	index, err := strconv.Atoi(indexStr)
	if err != nil {
		utils.JSONResponse(w, http.StatusBadRequest, utils.Payload{
			Success: false,
			Message: "Invalid index",
		})
		return
	}

	db := repositories.DB
	var transfer models.Transfer

	// Fetch transfer
	err = db.Where("token = ? AND deleted = ?", token, false).First(&transfer).Error
	if err != nil {
		utils.JSONResponse(w, http.StatusNotFound, utils.Payload{
			Success: false,
			Message: "Invalid or expired share link",
		})
		return
	}

	// Check expiry
	if time.Now().After(transfer.ExpiresAt) {
		utils.JSONResponse(w, http.StatusGone, utils.Payload{
			Success: false,
			Message: "This link has expired",
		})
		return
	}

	// Fetch the file by index in this transfer
	var file models.File
	err = db.Where("transfer_id = ? AND \"index\" = ? AND deleted = ?", transfer.ID, index, false).
		First(&file).Error
	if err != nil {
		utils.JSONResponse(w, http.StatusNotFound, utils.Payload{
			Success: false,
			Message: "File not found",
		})
		return
	}

	url, err := repositories.GeneratePresignedGetURL(r.Context(), file.Path, 15*time.Minute)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, utils.Payload{
			Success: false,
			Message: "Failed to generate download URL",
		})
		return
	}

	utils.JSONResponse(w, http.StatusOK, utils.Payload{
		Success: true,
		Message: "Presigned download URL generated successfully",
		Data: map[string]any{
			"url":          url,
			"content_type": file.ContentType,
			"filename":     file.Filename,
		},
	})
}
