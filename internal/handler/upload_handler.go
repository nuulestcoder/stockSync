package handler

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/adapolytech/stock-manager/pkg/response"
	"github.com/adapolytech/stock-manager/pkg/storage"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	maxUploadSize = 10 << 20
	fileField     = "file"
	filesField    = "files"
)

type UploadHandler struct {
	store storage.Store
}

func NewUploadHandler(store storage.Store) *UploadHandler {
	return &UploadHandler{store: store}
}

func (h *UploadHandler) Single(c *gin.Context) {
	companyID, ok := companyIDFromContext(c)
	if !ok {
		return
	}
	file, err := c.FormFile(fileField)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "a file is required")
		return
	}
	uploaded, err := h.upload(c, companyID, file)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, http.StatusCreated, uploaded, "file uploaded successfully")
}

func (h *UploadHandler) Multiple(c *gin.Context) {
	companyID, ok := companyIDFromContext(c)
	if !ok {
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid multipart form")
		return
	}
	files := form.File[filesField]
	if len(files) == 0 {
		response.Error(c, http.StatusBadRequest, "at least one file is required")
		return
	}
	uploaded := make([]storage.File, 0, len(files))
	for _, file := range files {
		item, uploadErr := h.upload(c, companyID, file)
		if uploadErr != nil {
			response.Error(c, http.StatusBadRequest, uploadErr.Error())
			return
		}
		uploaded = append(uploaded, item)
	}
	response.Success(c, http.StatusCreated, uploaded, "files uploaded successfully")
}

func (h *UploadHandler) upload(c *gin.Context, companyID bson.ObjectID, file *multipart.FileHeader) (storage.File, error) {
	if file.Size <= 0 {
		return storage.File{}, fmt.Errorf("file %q is empty", file.Filename)
	}
	if file.Size > maxUploadSize {
		return storage.File{}, fmt.Errorf("file %q exceeds the 10 MB limit", file.Filename)
	}
	if filepath.Ext(file.Filename) == "" {
		return storage.File{}, fmt.Errorf("file %q must have an extension", file.Filename)
	}
	reader, err := file.Open()
	if err != nil {
		return storage.File{}, fmt.Errorf("open file %q: %w", file.Filename, err)
	}
	defer reader.Close()
	header := make([]byte, 512)
	read, err := reader.Read(header)
	if err != nil {
		return storage.File{}, fmt.Errorf("read file %q: %w", file.Filename, err)
	}
	contentType := http.DetectContentType(header[:read])
	if !isSupportedImage(contentType) {
		return storage.File{}, fmt.Errorf("file %q must be an image", file.Filename)
	}
	if _, err := reader.Seek(0, 0); err != nil {
		return storage.File{}, fmt.Errorf("rewind file %q: %w", file.Filename, err)
	}
	return h.store.Upload(c.Request.Context(), companyID, file.Filename, contentType, file.Size, reader)
}

func isSupportedImage(contentType string) bool {
	switch strings.ToLower(contentType) {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}
