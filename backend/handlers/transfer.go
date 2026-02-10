package handlers

import (
	"crypto/rand"
	"errors"
	"fmt"
	"flatnasgo-backend/config"
	"flatnasgo-backend/models"
	"flatnasgo-backend/utils"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Helper to ensure directories exist
func ensureDir(path string) {
	os.MkdirAll(path, 0755)
}

var errUploadPermission = errors.New("upload permission denied")
var errUploadIndex = errors.New("upload invalid index")

type DownloadClaims struct {
	Username string `json:"username"`
	Filename string `json:"filename"`
	jwt.RegisteredClaims
}

func getTransferDir() string {
	return filepath.Join(config.DocDir, "transfer")
}

func getTransferIndexFile() string {
	return filepath.Join(getTransferDir(), "index.json")
}

func getUploadsDir() string {
	return filepath.Join(getTransferDir(), "uploads")
}

func getUserUploadsDir(username string) string {
	return filepath.Join(getTransferDir(), "users", username, "uploads")
}

func isValidUploadID(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			continue
		}
		return false
	}
	return true
}

func GetTransferItems(c *gin.Context) {
	ensureDir(getUploadsDir())
	
	itemType := c.Query("type")
	limitStr := c.Query("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	var data models.TransferData
	utils.ReadJSON(getTransferIndexFile(), &data)
	if data.Items == nil {
		data.Items = []models.TransferItem{}
	}

	// Sort by timestamp desc
	sort.Slice(data.Items, func(i, j int) bool {
		return data.Items[i].Timestamp > data.Items[j].Timestamp
	})

	filtered := []models.TransferItem{}
	for _, item := range data.Items {
		if itemType == "photo" {
			if item.Type == "file" && item.File != nil && strings.HasPrefix(item.File.Type, "image/") {
				filtered = append(filtered, item)
			}
		} else if itemType == "file" {
			if item.Type == "file" {
				filtered = append(filtered, item)
			}
		} else if itemType == "text" {
			if item.Type == "text" {
				filtered = append(filtered, item)
			}
		} else {
			filtered = append(filtered, item)
		}
	}

	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "items": filtered})
}

func SendText(c *gin.Context) {
	var req struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	item := models.TransferItem{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:      "text",
		Content:   req.Text,
		Timestamp: time.Now().UnixMilli(),
		Sender:    c.GetString("username"),
	}

	// Lock and update index
	indexPath := getTransferIndexFile()
	
	var data models.TransferData
	utils.ReadJSON(indexPath, &data)
	
	if data.Items == nil {
		data.Items = []models.TransferItem{}
	}
	data.Items = append([]models.TransferItem{item}, data.Items...)

	if err := utils.WriteJSON(indexPath, data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "item": item})
}

type UploadSession struct {
	UploadID    string   `json:"uploadId"`
	Username    string   `json:"username"`
	FileKey     string   `json:"fileKey"`
	FileName    string   `json:"fileName"`
	Size        int64    `json:"size"`
	Mime        string   `json:"mime"`
	ChunkSize   int64    `json:"chunkSize"`
	TotalChunks int      `json:"totalChunks"`
	CreatedAt   int64    `json:"createdAt"`
	Uploaded    []int    `json:"uploaded"`
}

func UploadInit(c *gin.Context) {
	var req struct {
		FileName  string `json:"fileName"`
		Size      int64  `json:"size"`
		Mime      string `json:"mime"`
		FileKey   string `json:"fileKey"`
		ChunkSize int64  `json:"chunkSize"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	if req.ChunkSize <= 0 || req.Size <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chunk size or file size"})
		return
	}

	username := c.GetString("username")
	uploadId := fmt.Sprintf("%x", time.Now().UnixNano()) // Simple ID
	
	totalChunks := int((req.Size + req.ChunkSize - 1) / req.ChunkSize)
	
	session := UploadSession{
		UploadID:    uploadId,
		Username:    username,
		FileKey:     req.FileKey,
		FileName:    req.FileName,
		Size:        req.Size,
		Mime:        req.Mime,
		ChunkSize:   req.ChunkSize,
		TotalChunks: totalChunks,
		CreatedAt:   time.Now().UnixMilli(),
		Uploaded:    []int{},
	}

	userDir := getUserUploadsDir(username)
	ensureDir(userDir)
	sessionFile := filepath.Join(userDir, uploadId+".json")
	if err := utils.WriteJSON(sessionFile, session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize upload"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"uploadId":    uploadId,
		"chunkSize":   req.ChunkSize,
		"totalChunks": totalChunks,
		"uploaded":    []int{},
	})
}

func UploadChunk(c *gin.Context) {
	uploadId := c.PostForm("uploadId")
	indexStr := c.PostForm("index")
	
	if uploadId == "" || indexStr == "" || !isValidUploadID(uploadId) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing params"})
		return
	}
	
	index, err := strconv.Atoi(indexStr)
	if err != nil || index < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid index"})
		return
	}
	username := c.GetString("username")
	userDir := getUserUploadsDir(username)
	sessionFile := filepath.Join(userDir, uploadId+".json")
	
	var session UploadSession
	if err := utils.ReadJSON(sessionFile, &session); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}
	if session.Username != username {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}
	if index >= session.TotalChunks {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid index"})
		return
	}

	file, err := c.FormFile("chunk")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file"})
		return
	}

	chunkDir := filepath.Join(userDir, uploadId+"_chunks")
	ensureDir(chunkDir)
	chunkPath := filepath.Join(chunkDir, fmt.Sprintf("%d", index))
	
	if err := c.SaveUploadedFile(file, chunkPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Save failed"})
		return
	}

	err = utils.WithFileLock(sessionFile, func() error {
		var current UploadSession
		if err := utils.ReadJSONUnlocked(sessionFile, &current); err != nil {
			return err
		}
		if current.Username != username {
			return errUploadPermission
		}
		if index >= current.TotalChunks {
			return errUploadIndex
		}
		uploaded := make(map[int]struct{}, len(current.Uploaded)+1)
		for _, v := range current.Uploaded {
			uploaded[v] = struct{}{}
		}
		uploaded[index] = struct{}{}
		current.Uploaded = current.Uploaded[:0]
		for v := range uploaded {
			current.Uploaded = append(current.Uploaded, v)
		}
		sort.Ints(current.Uploaded)
		return utils.WriteJSONUnlocked(sessionFile, current)
	})
	if err != nil {
		if errors.Is(err, errUploadPermission) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
			return
		}
		if errors.Is(err, errUploadIndex) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid index"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func UploadComplete(c *gin.Context) {
	var req struct {
		UploadId string `json:"uploadId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	if !isValidUploadID(req.UploadId) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid upload ID"})
		return
	}

	username := c.GetString("username")
	userDir := getUserUploadsDir(username)
	sessionFile := filepath.Join(userDir, req.UploadId+".json")
	
	var session UploadSession
	if err := utils.ReadJSON(sessionFile, &session); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}
	if session.Username != username {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}
	if session.TotalChunks <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid upload session"})
		return
	}

	// Assemble
	chunkDir := filepath.Join(userDir, req.UploadId+"_chunks")
	
	// Use random filename to prevent guessing
	randBytes := make([]byte, 16)
	if _, err := rand.Read(randBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize upload"})
		return
	}
	finalName := fmt.Sprintf("%x%s", randBytes, filepath.Ext(session.FileName))
	
	finalPath := filepath.Join(getUploadsDir(), finalName)
	ensureDir(getUploadsDir())

	outFile, err := os.Create(finalPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Create file failed"})
		return
	}
	defer outFile.Close()

	for i := 0; i < session.TotalChunks; i++ {
		chunkPath := filepath.Join(chunkDir, fmt.Sprintf("%d", i))
		in, err := os.Open(chunkPath)
		if err != nil {
			outFile.Close()
			os.Remove(finalPath)
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Missing chunk %d", i)})
			return
		}
		_, err = io.Copy(outFile, in)
		in.Close()
		if err != nil {
			outFile.Close()
			os.Remove(finalPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assemble file"})
			return
		}
	}

	// Cleanup
	os.RemoveAll(chunkDir)
	os.Remove(sessionFile)

	// Add to index
	item := models.TransferItem{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:      "file",
		Timestamp: time.Now().UnixMilli(),
		Sender:    username,
		File: &models.TransferFile{
			Name: session.FileName,
			Size: session.Size,
			Type: session.Mime,
			Url:  "/api/transfer/file/" + finalName,
		},
	}

	var data models.TransferData
	utils.ReadJSON(getTransferIndexFile(), &data)
	if data.Items == nil {
		data.Items = []models.TransferItem{}
	}
	data.Items = append([]models.TransferItem{item}, data.Items...)
	if err := utils.WriteJSON(getTransferIndexFile(), data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "item": item})
}

func DownloadToken(c *gin.Context) {
	var body struct {
		Url string `json:"url"`
	}
	_ = c.ShouldBindJSON(&body)
	u, err := url.Parse(body.Url)
	if err != nil || u.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid url"})
		return
	}
	username := c.GetString("username")
	if username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	name := filepath.Base(u.Path)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid url"})
		return
	}
	if _, err := os.Stat(filepath.Join(getUploadsDir(), name)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	claims := DownloadClaims{
		Username: username,
		Filename: name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "download",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(config.GetSecretKeyString()))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to issue token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "token": signed})
}

func DeleteItem(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing ID"})
		return
	}

	username := c.GetString("username")
	if username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var data models.TransferData
	utils.ReadJSON(getTransferIndexFile(), &data)
	
	newList := []models.TransferItem{}
	var deletedItem *models.TransferItem
	for _, item := range data.Items {
		if item.ID == id {
			// IDOR Check
			if item.Sender != username && username != "admin" {
				c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
				return
			}
			deletedItem = &item
			continue
		}
		newList = append(newList, item)
	}
	
	if deletedItem != nil {
		data.Items = newList
		if err := utils.WriteJSON(getTransferIndexFile(), data); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save"})
			return
		}
		
		// Delete file if needed
		if deletedItem.Type == "file" && deletedItem.File != nil {
			filename := filepath.Base(deletedItem.File.Url)
			os.Remove(filepath.Join(getUploadsDir(), filename))
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func ServeFile(c *gin.Context) {
	filename := filepath.Base(c.Param("filename"))
	if filename == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid filename"})
		return
	}
	tokenStr := c.Query("token")
	if tokenStr != "" {
		claims := &DownloadClaims{}
		tok, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.GetSecretKeyString()), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		if err != nil || tok == nil || !tok.Valid || claims.Filename != filename {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
	} else {
		if c.GetString("username") == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
	}
	path := filepath.Join(getUploadsDir(), filename)
	c.File(path)
}
