package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/google/uuid"
	"github.com/rohits-web03/obscyra/internal/models"
	"github.com/rohits-web03/obscyra/internal/repositories"
	"github.com/rohits-web03/obscyra/internal/utils"
	"gorm.io/gorm"
)

type PresignInput []struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

type PresignedFile struct {
	Filename  string `json:"filename"`
	UploadURL string `json:"uploadURL"`
	Key       string `json:"key"`
}

type PresignResponse struct {
	Token string          `json:"token"`
	URLs  []PresignedFile `json:"urls"`
}

type CompleteUploadInput struct {
	Token string `json:"token"`
	Files []struct {
		Filename    string `json:"filename"`
		Size        int64  `json:"size"`
		Key         string `json:"key"`
		ContentType string `json:"contentType"`
	} `json:"files"`
}

const maxUploadSize = 100 << 20 // 100 MB

// POST /api/v1/files/presign
// PresignUpload generates presigned URLs for uploading files to R2 storage.
// @Summary Generate presigned URLs for file upload
// @Description Accepts a list of files (name and size), validates the total size, and returns presigned PUT URLs for each file. Each upload session is identified by a unique token.
// @Tags Files
// @Accept json
// @Produce json
// @Param input body PresignInput true "List of files to upload"
// @Success 200 {object} utils.Payload{data=PresignResponse} "Presigned URLs generated successfully"
// @Failure 400 {object} utils.Payload "Invalid input or size limit exceeded"
// @Failure 405 {object} utils.Payload "Method not allowed"
// @Failure 500 {object} utils.Payload "Failed to generate presigned URL"
// @Router /api/v1/files/presign [post]
func PresignUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, utils.Payload{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	var input PresignInput

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&input); err != nil {
		utils.JSONResponse(w, http.StatusBadRequest, utils.Payload{
			Success: false,
			Message: "Invalid input",
		})
		return
	}

	// Calculate total size
	var totalSize int64
	for _, f := range input {
		totalSize += f.Size
	}
	if totalSize > maxUploadSize {
		utils.JSONResponse(w, http.StatusBadRequest, utils.Payload{
			Success: false,
			Message: "Total file size exceeds 100 MB limit",
		})
		return
	}

	token, err := utils.GenerateSecureToken(32)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, utils.Payload{
			Success: false,
			Message: "Failed to create transfer token",
		})
		return
	}

	results := make([]PresignedFile, 0, len(input))

	for _, f := range input {
		key := "uploads/" + token + "/" + uuid.New().String() + "_" + f.Filename
		uploadURL, err := repositories.GeneratePresignedPutURL(context.Background(), key, 15*time.Minute)
		if err != nil {
			utils.JSONResponse(w, http.StatusInternalServerError, utils.Payload{
				Success: false,
				Message: "Failed to generate presigned URL",
			})
			return
		}
		results = append(results, PresignedFile{
			Filename:  f.Filename,
			UploadURL: uploadURL,
			Key:       key,
		})
	}

	utils.JSONResponse(w, http.StatusOK, utils.Payload{
		Success: true,
		Message: "Presigned URLs generated successfully",
		Data: map[string]any{
			"token": token,
			"urls":  results,
		},
	})
}

// POST /api/v1/files/complete
// CompleteUpload finalizes an anonymous upload and stores metadata in the database.
// @Summary Complete file upload
// @Description Verifies uploaded files on R2, stores file metadata, and registers the upload session in the database. Each transfer is valid for 1 hour and limited to 100MB for anonymous uploads.
// @Tags Files
// @Accept json
// @Produce json
// @Param input body CompleteUploadInput true "Upload completion payload"
// @Success 200 {object} utils.Payload{data=map[string]interface{}} "Files uploaded successfully"
// @Failure 400 {object} utils.Payload "Invalid input or verification failed"
// @Failure 405 {object} utils.Payload "Method not allowed"
// @Failure 500 {object} utils.Payload "Database error"
// @Router /api/v1/files/complete [post]
func CompleteUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, utils.Payload{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	var input CompleteUploadInput

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&input); err != nil {
		utils.JSONResponse(w, http.StatusBadRequest, utils.Payload{
			Success: false,
			Message: "Invalid input",
		})
		return
	}

	if input.Token == "" || len(input.Files) == 0 {
		utils.JSONResponse(w, http.StatusBadRequest, utils.Payload{
			Success: false,
			Message: "Missing token or no files provided",
		})
		return
	}

	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)

	for _, f := range input.Files {
		file := f
		g.Go(func() error {
			exists, err := repositories.VerifyObjectExists(ctx, file.Key)
			if err != nil {
				return fmt.Errorf("failed to verify %s: %w", file.Filename, err)
			}
			if !exists {
				return fmt.Errorf("file not found: %s", file.Filename)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		utils.JSONResponse(w, http.StatusBadRequest, utils.Payload{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	var TotalSize int64
	for _, f := range input.Files {
		TotalSize += f.Size
	}

	if TotalSize > maxUploadSize {
		utils.JSONResponse(w, http.StatusBadRequest, utils.Payload{
			Success: false,
			Message: "Total upload size exceeds 100MB limit for anonymous transfers",
		})
		return
	}

	// Begin DB transaction
	db := repositories.DB
	err := db.Transaction(func(tx *gorm.DB) error {
		// Create transfer record
		transfer := models.Transfer{
			Token:       input.Token,
			ExpiresAt:   time.Now().Add(1 * time.Hour),
			IsAnonymous: true,
			TotalSize:   TotalSize,
		}

		if err := tx.Create(&transfer).Error; err != nil {
			return err
		}

		// Store each file
		for i, f := range input.Files {
			file := models.File{
				TransferID:  transfer.ID,
				Filename:    f.Filename,
				Size:        f.Size,
				Path:        f.Key,
				ContentType: f.ContentType,
				Index:       i,
			}
			if err := tx.Create(&file).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, utils.Payload{
			Success: false,
			Message: "Failed to store files in DB",
		})
		return
	}

	utils.JSONResponse(w, http.StatusOK, utils.Payload{
		Success: true,
		Message: "Files uploaded successfully",
		Data: map[string]interface{}{
			"share_code": input.Token,
			"expires_in": "1h",
		},
	})
}
