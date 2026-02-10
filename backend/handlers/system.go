package handlers

import (
	"encoding/json"
	"flatnasgo-backend/config"
	"flatnasgo-backend/utils"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

func GetSystemStats(c *gin.Context) {
	v, _ := mem.VirtualMemory()
	cStats, _ := cpu.Info()
	percent, _ := cpu.Percent(0, false)

	// Use the volume of BaseDir
	volume := filepath.VolumeName(config.BaseDir)
	if volume == "" {
		volume = "/"
	} else {
		volume = volume + "\\"
	}
	d, _ := disk.Usage(volume)
	n, _ := net.IOCounters(false)
	h, _ := host.Info()

	cpuLoad := 0.0
	if len(percent) > 0 {
		cpuLoad = percent[0]
	}

	brand := "Unknown"
	speed := 0.0
	if len(cStats) > 0 {
		brand = cStats[0].ModelName
		speed = cStats[0].Mhz / 1000.0
	}

	data := gin.H{
		"cpu": gin.H{
			"currentLoad": cpuLoad,
			"cores":       runtime.NumCPU(),
			"brand":       brand,
			"speed":       speed,
		},
		"mem": gin.H{
			"total":     v.Total,
			"used":      v.Used,
			"active":    v.Active,
			"available": v.Available,
		},
		"disk": []gin.H{
			{
				"fs":    d.Fstype,
				"type":  "Fixed",
				"size":  d.Total,
				"used":  d.Used,
				"use":   d.UsedPercent,
				"mount": d.Path,
			},
		},
		"network": n,
		"os": gin.H{
			"distro":   h.Platform,
			"release":  h.PlatformVersion,
			"hostname": h.Hostname,
			"arch":     h.KernelArch,
		},
		"uptime": h.Uptime,
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func GetCustomScripts(c *gin.Context) {
	username := c.GetString("username")
	if username == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"css":     []interface{}{},
			"js":      []interface{}{},
		})
		return
	}
	path := filepath.Join(config.DataDir, "custom_scripts.json")
	payload := CustomScriptsPayload{
		CSS: []interface{}{},
		JS:  []interface{}{},
	}
	var data map[string]CustomScriptsPayload
	if err := utils.ReadJSON(path, &data); err == nil {
		if entry, ok := data[username]; ok {
			payload = entry
			if payload.CSS == nil {
				payload.CSS = []interface{}{}
			}
			if payload.JS == nil {
				payload.JS = []interface{}{}
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"css":     payload.CSS,
		"js":      payload.JS,
	})
}

type CustomScriptsPayload struct {
	CSS []interface{} `json:"css"`
	JS  []interface{} `json:"js"`
}

func SaveCustomScripts(c *gin.Context) {
	username := c.GetString("username")
	if username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var payload CustomScriptsPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	if payload.CSS == nil {
		payload.CSS = []interface{}{}
	}
	if payload.JS == nil {
		payload.JS = []interface{}{}
	}
	path := filepath.Join(config.DataDir, "custom_scripts.json")
	var data map[string]CustomScriptsPayload
	if err := utils.ReadJSON(path, &data); err != nil || data == nil {
		data = make(map[string]CustomScriptsPayload)
	}
	data[username] = payload
	if err := utils.WriteJSON(path, data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save custom scripts"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// IPCache holds the cached public IP information
type IPCache struct {
	IP       string
	Location string
	Updated  time.Time
	Mutex    sync.RWMutex
}

var globalIPCache IPCache
var isFetchingIP int32

// StartIPFetcher starts a background goroutine to fetch public IP every 6 hours
func StartIPFetcher() {
	// Fetch immediately on start
	go func() {
		fetchIPAndCache()
		ticker := time.NewTicker(6 * time.Hour)
		for range ticker.C {
			fetchIPAndCache()
		}
	}()
}

func fetchIPAndCache() bool {
	if !atomic.CompareAndSwapInt32(&isFetchingIP, 0, 1) {
		return false
	}
	defer atomic.StoreInt32(&isFetchingIP, 0)

	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get("http://ip-api.com/json/?lang=zh-CN")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return false
	}

	if status, ok := result["status"].(string); ok && status == "fail" {
		return false
	}

	globalIPCache.Mutex.Lock()
	defer globalIPCache.Mutex.Unlock()

	if query, ok := result["query"].(string); ok {
		globalIPCache.IP = query
	}
	globalIPCache.Location = getLocationString(result)
	globalIPCache.Updated = time.Now()
	return true
}

func GetIP(c *gin.Context) {
	refresh := strings.TrimSpace(c.Query("refresh"))
	refreshed := false
	if refresh == "1" || strings.EqualFold(refresh, "true") {
		fetchIPAndCache()
		refreshed = true
	}

	globalIPCache.Mutex.RLock()
	ip := globalIPCache.IP
	location := globalIPCache.Location
	globalIPCache.Mutex.RUnlock()

	if ip != "" {
		c.JSON(http.StatusOK, gin.H{
			"success":        true,
			"ip":             ip,
			"location":       location,
			"clientIp":       c.ClientIP(),
			"clientIpSource": "header",
			"cached":         true,
		})
		return
	}

	// If we just tried to refresh and failed (ip is still empty), don't try again immediately
	if refreshed {
		c.JSON(http.StatusOK, gin.H{
			"success":        false,
			"ip":             c.ClientIP(),
			"clientIp":       c.ClientIP(),
			"clientIpSource": "request",
		})
		return
	}

	// Try to fetch from external API (Fallback if cache is empty and we haven't just tried)
	// ip-api.com is free for non-commercial use
	client := http.Client{
		Timeout: 4 * time.Second,
	}
	resp, err := client.Get("http://ip-api.com/json/?lang=zh-CN")
	if err != nil {
		// Fallback to client IP
		c.JSON(http.StatusOK, gin.H{
			"success":        false,
			"ip":             c.ClientIP(),
			"clientIp":       c.ClientIP(),
			"clientIpSource": "request",
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success":        false,
			"ip":             c.ClientIP(),
			"clientIp":       c.ClientIP(),
			"clientIpSource": "request",
		})
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success":        false,
			"ip":             c.ClientIP(),
			"clientIp":       c.ClientIP(),
			"clientIpSource": "request",
		})
		return
	}

	// Update cache since we fetched it
	if status, ok := result["status"].(string); ok && status != "fail" {
		globalIPCache.Mutex.Lock()
		if query, ok := result["query"].(string); ok {
			globalIPCache.IP = query
		}
		globalIPCache.Location = getLocationString(result)
		globalIPCache.Updated = time.Now()
		globalIPCache.Mutex.Unlock()
	}

	// Format response to match frontend expectations
	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"ip":             result["query"],
		"location":       getLocationString(result),
		"clientIp":       c.ClientIP(),
		"clientIpSource": "header",
	})
}

func getLocationString(data map[string]interface{}) string {
	parts := []string{}
	if country, ok := data["country"].(string); ok {
		parts = append(parts, country)
	}
	if region, ok := data["regionName"].(string); ok {
		parts = append(parts, region)
	}
	if city, ok := data["city"].(string); ok {
		parts = append(parts, city)
	}
	if isp, ok := data["isp"].(string); ok {
		parts = append(parts, isp)
	}
	return strings.Join(parts, " ")
}

// Ping handles latency check
func Ping(c *gin.Context) {
	target := c.Query("target")
	if target == "" {
		target = "223.5.5.5"
	}

	// Ping implementation based on OS
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// -n 1: count 1
		// -w 1000: timeout 1000ms
		cmd = exec.Command("ping", "-n", "1", "-w", "1000", target)
	} else {
		// Linux/Unix
		// -c 1: count 1
		// -W 1: timeout 1 second
		cmd = exec.Command("ping", "-c", "1", "-W", "1", target)
	}
	output, err := cmd.CombinedOutput()

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "Ping failed",
		})
		return
	}

	outStr := string(output)
	// Look for time=XXms
	// Windows output: "Reply from ... time=12ms ..."
	// Linux output: "... time=12.3 ms"
	// Chinese output: "来自 ... 时间=12ms ..."
	// Regex to capture digits and optional decimals, allowing optional space before ms
	// Modified to be more permissive for Windows GBK output (ignoring the "time" label which might be garbled)
	re := regexp.MustCompile(`[=<]([\d\.]+) ?ms`)
	matches := re.FindStringSubmatch(outStr)

	if len(matches) > 1 {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"latency": matches[1] + "ms",
		})
	} else {
		// Try to handle "0ms" or "<1ms"
		if strings.Contains(outStr, "<1ms") {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"latency": "<1ms",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "Could not parse latency",
		})
	}
}

// GetMusicList returns list of music files
func GetMusicList(c *gin.Context) {
	var files []string
	err := filepath.Walk(config.MusicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".mp3" || ext == ".flac" || ext == ".wav" || ext == ".m4a" || ext == ".ogg" {
				rel, _ := filepath.Rel(config.MusicDir, path)
				// Convert windows path separator to forward slash for web url
				rel = strings.ReplaceAll(rel, "\\", "/")
				files = append(files, rel)
			}
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, []string{})
		return
	}

	c.JSON(http.StatusOK, files)
}

// RTT handles simple round-trip time check
func RTT(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"time":    time.Now().UnixNano(),
	})
}
