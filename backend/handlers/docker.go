package handlers

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"flatnasgo-backend/config"
	"flatnasgo-backend/models"
	"flatnasgo-backend/utils"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
)

var dockerClient *client.Client
var dockerHostUsed string
var dockerInitError error
var updateStatus UpdateCheckStatus
var updateStatusMu sync.RWMutex

func InitDocker() {
	if !dockerEnabled() {
		dockerClient = nil
		dockerHostUsed = ""
		dockerInitError = nil
		return
	}
	host := resolveDockerHost()
	opts := []client.Opt{client.WithAPIVersionNegotiation()}
	if host == "" {
		opts = append(opts, client.FromEnv)
	} else {
		opts = append(opts, client.WithHost(host))
	}
	var err error
	dockerClient, err = client.NewClientWithOpts(opts...)
	dockerHostUsed = host
	if err != nil {
		log.Printf("Failed to init docker client: %v", err)
		dockerClient = nil
		dockerInitError = err
	} else {
		dockerInitError = nil
	}
}

func dockerEnabled() bool {
	var sysConfig models.SystemConfig
	utils.ReadJSON(config.SystemConfigFile, &sysConfig)
	return sysConfig.EnableDocker
}

func resolveDockerHost() string {
	var sysConfig models.SystemConfig
	utils.ReadJSON(config.SystemConfigFile, &sysConfig)
	if strings.TrimSpace(sysConfig.DockerHost) != "" {
		return normalizeDockerHost(sysConfig.DockerHost)
	}
	if env := strings.TrimSpace(os.Getenv("DOCKER_HOST")); env != "" {
		return normalizeDockerHost(env)
	}
	return defaultDockerHost()
}

func normalizeDockerHost(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return ""
	}
	lower := strings.ToLower(v)
	if strings.HasPrefix(lower, "unix://") ||
		strings.HasPrefix(lower, "npipe://") ||
		strings.HasPrefix(lower, "tcp://") ||
		strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") {
		return v
	}
	if strings.HasPrefix(v, "//./pipe/") {
		return "npipe:////./pipe/" + strings.TrimPrefix(v, "//./pipe/")
	}
	if strings.HasPrefix(v, `\\.\pipe\`) {
		return "npipe:////./pipe/" + strings.TrimPrefix(v, `\\.\pipe\`)
	}
	if strings.HasPrefix(v, "/") {
		return "unix://" + v
	}
	if runtime.GOOS == "windows" && strings.Contains(v, "pipe/docker_engine") {
		return "npipe:////./pipe/docker_engine"
	}
	if strings.Count(v, ":") == 1 && !strings.Contains(v, "://") {
		return "tcp://" + v
	}
	return v
}

func defaultDockerHost() string {
	switch runtime.GOOS {
	case "windows":
		return "npipe:////./pipe/docker_engine"
	case "linux", "darwin":
		return "unix:///var/run/docker.sock"
	default:
		return ""
	}
}

func getDockerClient() *client.Client {
	if !dockerEnabled() {
		return nil
	}
	host := resolveDockerHost()
	if dockerClient == nil || host != dockerHostUsed {
		InitDocker()
	}
	return dockerClient
}

type UpdateFailure struct {
	Name  string `json:"name"`
	Error string `json:"error"`
}

type UpdateCheckStatus struct {
	LastCheck    int64           `json:"lastCheck"`
	IsChecking   bool            `json:"isChecking"`
	LastError    string          `json:"lastError"`
	CheckedCount int             `json:"checkedCount"`
	TotalCount   int             `json:"totalCount,omitempty"`
	UpdateCount  int             `json:"updateCount"`
	Failures     []UpdateFailure `json:"failures,omitempty"`
}

func ListContainers(c *gin.Context) {
	if !dockerEnabled() {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "Docker not available", "data": []interface{}{}})
		return
	}
	dc := getDockerClient()
	if dc == nil {
		errMsg := "Docker not available"
		if dockerInitError != nil {
			errMsg = dockerInitError.Error()
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "error": errMsg, "data": []interface{}{}})
		return
	}

	containers, err := dc.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error(), "data": []interface{}{}})
		return
	}

	updateStatusMu.RLock()
	us := updateStatus
	updateStatusMu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"success": true, "data": containers, "updateStatus": us})
}

func GetDockerStatus(c *gin.Context) {
	if !dockerEnabled() {
		c.JSON(http.StatusOK, gin.H{
			"hasUpdate": false,
		})
		return
	}
	updateStatusMu.RLock()
	hasUpdate := updateStatus.UpdateCount > 0
	updateStatusMu.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"hasUpdate": hasUpdate,
	})
}

func ContainerAction(c *gin.Context) {
	username := c.GetString("username")
	if username != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	id := c.Param("id")
	action := c.Param("action")

	if !dockerEnabled() {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Docker not available"})
		return
	}
	dc := getDockerClient()
	if dc == nil {
		errMsg := "Docker not available"
		if dockerInitError != nil {
			errMsg = dockerInitError.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg})
		return
	}

	ctx := context.Background()
	var err error

	switch action {
	case "start":
		err = dc.ContainerStart(ctx, id, container.StartOptions{})
	case "stop":
		err = dc.ContainerStop(ctx, id, container.StopOptions{})
	case "restart":
		err = dc.ContainerRestart(ctx, id, container.StopOptions{})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func GetDockerInfo(c *gin.Context) {
	if !dockerEnabled() {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "Docker not available", "socketPath": resolveDockerHost()})
		return
	}
	dc := getDockerClient()
	if dc == nil {
		errMsg := "Docker not available"
		if dockerInitError != nil {
			errMsg = dockerInitError.Error()
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "error": errMsg, "socketPath": resolveDockerHost()})
		return
	}
	info, err := dc.Info(context.Background())
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error(), "socketPath": dc.DaemonHost()})
		return
	}
	version, err := dc.ServerVersion(context.Background())
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "info": info, "version": types.Version{}, "socketPath": dc.DaemonHost()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "info": info, "version": version, "socketPath": dc.DaemonHost()})
}

func GetDockerDebug(c *gin.Context) {
	var sysConfig models.SystemConfig
	utils.ReadJSON(config.SystemConfigFile, &sysConfig)

	dockerHostRaw := strings.TrimSpace(sysConfig.DockerHost)
	dockerHostEnv := strings.TrimSpace(os.Getenv("DOCKER_HOST"))
	dockerHostResolved := resolveDockerHost()

	enabled := sysConfig.EnableDocker
	dc := getDockerClient()
	clientAvailable := dc != nil
	daemonHost := ""
	pingOk := false
	pingError := ""
	if dc != nil {
		daemonHost = dc.DaemonHost()
		if _, err := dc.Ping(context.Background()); err != nil {
			pingError = err.Error()
		} else {
			pingOk = true
		}
	}

	initError := ""
	if dockerInitError != nil {
		initError = dockerInitError.Error()
	}

	c.JSON(http.StatusOK, gin.H{
		"enableDocker":       enabled,
		"dockerHostRaw":      dockerHostRaw,
		"dockerHostEnv":      dockerHostEnv,
		"dockerHostResolved": dockerHostResolved,
		"clientAvailable":    clientAvailable,
		"daemonHost":         daemonHost,
		"pingOk":             pingOk,
		"pingError":          pingError,
		"initError":          initError,
	})
}

type InspectLite struct {
	NetworkMode string `json:"networkMode"`
	Ports       []int  `json:"ports"`
}

func ContainerInspectLite(c *gin.Context) {
	if !dockerEnabled() {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "Docker not available"})
		return
	}
	dc := getDockerClient()
	if dc == nil {
		errMsg := "Docker not available"
		if dockerInitError != nil {
			errMsg = dockerInitError.Error()
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "error": errMsg})
		return
	}
	id := c.Param("id")
	ctx := context.Background()
	inspected, err := dc.ContainerInspect(ctx, id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error()})
		return
	}
	networkMode := string(inspected.HostConfig.NetworkMode)
	ports := []int{}
	if inspected.NetworkSettings != nil && inspected.NetworkSettings.Ports != nil {
		for _, bindings := range inspected.NetworkSettings.Ports {
			for _, b := range bindings {
				if b.HostPort != "" {
					if v, err := strconv.Atoi(b.HostPort); err == nil && v > 0 && v <= 65535 {
						ports = append(ports, v)
					}
				}
			}
		}
	}
	if len(ports) == 0 && strings.EqualFold(networkMode, "host") {
		if inspected.Config != nil {
			for p := range inspected.Config.ExposedPorts {
				s := string(p)
				if idx := strings.IndexByte(s, '/'); idx > 0 {
					s = s[:idx]
				}
				if v, err := strconv.Atoi(s); err == nil && v > 0 && v <= 65535 {
					ports = append(ports, v)
				}
			}
		}
	}
	data := InspectLite{NetworkMode: networkMode, Ports: ports}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func TriggerUpdateCheck(c *gin.Context) {
	username := c.GetString("username")
	if username != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}
	if !dockerEnabled() {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "Docker not available"})
		return
	}
	dc := getDockerClient()
	if dc == nil {
		errMsg := "Docker not available"
		if dockerInitError != nil {
			errMsg = dockerInitError.Error()
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "error": errMsg})
		return
	}
	updateStatusMu.Lock()
	if updateStatus.IsChecking {
		updateStatusMu.Unlock()
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}
	updateStatus.IsChecking = true
	updateStatus.LastError = ""
	updateStatus.CheckedCount = 0
	updateStatus.UpdateCount = 0
	updateStatus.Failures = nil
	updateStatusMu.Unlock()
	go func() {
		ctx := context.Background()
		list, err := dc.ContainerList(ctx, container.ListOptions{All: false})
		total := 0
		if err == nil {
			total = len(list)
		}
		updateStatusMu.Lock()
		updateStatus.TotalCount = total
		updateStatusMu.Unlock()
		if err != nil {
			updateStatusMu.Lock()
			updateStatus.LastError = err.Error()
			updateStatus.IsChecking = false
			updateStatus.LastCheck = time.Now().UnixMilli()
			updateStatusMu.Unlock()
			return
		}
		for _, ctn := range list {
			updateStatusMu.Lock()
			updateStatus.CheckedCount++
			updateStatusMu.Unlock()

			imageRef, ok := resolveTaggedImageRef(ctn.Image)
			if !ok {
				continue
			}
			pullCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			rc, err := dc.ImagePull(pullCtx, imageRef, types.ImagePullOptions{})
			cancel()
			if err != nil {
				addUpdateFailure(ctn, err)
				continue
			}
			_, _ = io.Copy(io.Discard, rc)
			_ = rc.Close()

			inspected, _, err := dc.ImageInspectWithRaw(ctx, imageRef)
			if err != nil {
				addUpdateFailure(ctn, err)
				continue
			}
			if inspected.ID != "" && inspected.ID != ctn.ImageID {
				updateStatusMu.Lock()
				updateStatus.UpdateCount++
				updateStatusMu.Unlock()
			}
		}
		updateStatusMu.Lock()
		updateStatus.IsChecking = false
		updateStatus.LastCheck = time.Now().UnixMilli()
		updateStatusMu.Unlock()
	}()
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func resolveTaggedImageRef(image string) (string, bool) {
	image = strings.TrimSpace(image)
	if image == "" {
		return "", false
	}
	if strings.Contains(image, "@") {
		return "", false
	}
	lastColon := strings.LastIndex(image, ":")
	lastSlash := strings.LastIndex(image, "/")
	if lastColon > lastSlash {
		return image, true
	}
	return "", false
}

func addUpdateFailure(ctn types.Container, err error) {
	name := ctn.ID
	if len(ctn.Names) > 0 {
		name = strings.TrimPrefix(ctn.Names[0], "/")
	}
	updateStatusMu.Lock()
	updateStatus.Failures = append(updateStatus.Failures, UpdateFailure{
		Name:  name,
		Error: err.Error(),
	})
	updateStatusMu.Unlock()
}
