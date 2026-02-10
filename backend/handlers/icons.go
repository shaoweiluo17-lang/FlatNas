package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// AliIcons Cache
type cachedAliIcons struct {
	Data      interface{}
	Timestamp time.Time
}

var (
	aliIconsCache cachedAliIcons
	aliIconsMutex sync.RWMutex
	// Cache duration: 24 hours
	aliIconsCacheDuration = 24 * time.Hour
)

const (
	// Use the URL that we verified works
	aliIconsURL = "https://icon-manager.1851365c.er.aliyun-esa.net/icons.json"
)

// GetAliIcons proxies the request to Alibaba Icon Manager to avoid CORS issues
func GetAliIcons(c *gin.Context) {
	aliIconsMutex.RLock()
	if aliIconsCache.Data != nil && time.Since(aliIconsCache.Timestamp) < aliIconsCacheDuration {
		data := aliIconsCache.Data
		aliIconsMutex.RUnlock()
		c.JSON(http.StatusOK, data)
		return
	}
	aliIconsMutex.RUnlock()

	// Fetch from upstream
	resp, err := http.Get(aliIconsURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch icons from upstream", "details": err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Upstream returned non-200 status", "status": resp.StatusCode})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response body", "details": err.Error()})
		return
	}

	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse JSON", "details": err.Error()})
		return
	}

	// Update cache
	aliIconsMutex.Lock()
	aliIconsCache.Data = data
	aliIconsCache.Timestamp = time.Now()
	aliIconsMutex.Unlock()

	c.JSON(http.StatusOK, data)
}

// GetIconBase64 fetches a URL and returns it as base64
func GetIconBase64(c *gin.Context) {
	urlStr := c.Query("url")
	if urlStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing url parameter"})
		return
	}

	resp, err := http.Get(urlStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch icon", "details": err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Upstream returned non-200 status", "status": resp.StatusCode})
		return
	}

	// Limit size to avoid memory issues (e.g., 5MB)
	const maxLimit = 5 * 1024 * 1024
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxLimit))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read body", "details": err.Error()})
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream" // fallback
	}

	base64Str := base64.StdEncoding.EncodeToString(body)
	dataURI := fmt.Sprintf("data:%s;base64,%s", contentType, base64Str)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"icon":    dataURI,
	})
}
