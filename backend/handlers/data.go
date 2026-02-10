package handlers

import (
	"flatnasgo-backend/config"
	"flatnasgo-backend/models"
	"flatnasgo-backend/utils"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func GetData(c *gin.Context) {
	username := c.GetString("username")
	isGuest := false
	if username == "" {
		username = "admin"
		isGuest = true
	}

	var sysConfig models.SystemConfig
	utils.ReadJSON(config.SystemConfigFile, &sysConfig)

	userFile := filepath.Join(config.UsersDir, username+".json")
	if username == "admin" && sysConfig.AuthMode == "single" {
		userFile = filepath.Join(config.DataDir, "data.json")
	}

	// Use map[string]interface{} to preserve all fields
	var userData map[string]interface{}
	if err := utils.ReadJSON(userFile, &userData); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User data not found"})
		return
	}

	// Remove password from response
	delete(userData, "password")

	if isGuest {
		// Filter public items manually in the map structure
		// This is tricky with untyped map, but necessary to preserve data integrity
		if groups, ok := userData["groups"].([]interface{}); ok {
			var filteredGroups []interface{}
			for _, g := range groups {
				if groupMap, ok := g.(map[string]interface{}); ok {
					if items, ok := groupMap["items"].([]interface{}); ok {
						var publicItems []interface{}
						for _, item := range items {
							if itemMap, ok := item.(map[string]interface{}); ok {
								if isPublic, ok := itemMap["isPublic"].(bool); ok && isPublic {
									publicItems = append(publicItems, itemMap)
								}
							}
						}
						// Only keep group if it has public items (or maybe keep empty groups?)
						// Previous logic: if len(publicItems) > 0 { ... }
						if len(publicItems) > 0 {
							groupMap["items"] = publicItems
							filteredGroups = append(filteredGroups, groupMap)
						}
					}
				}
			}
			userData["groups"] = filteredGroups
		}

		if widgets, ok := userData["widgets"].([]interface{}); ok {
			var filteredWidgets []interface{}
			for _, w := range widgets {
				if widgetMap, ok := w.(map[string]interface{}); ok {
					if isPublic, ok := widgetMap["isPublic"].(bool); ok && isPublic {
						filteredWidgets = append(filteredWidgets, widgetMap)
					}
				}
			}
			userData["widgets"] = filteredWidgets
		}
	}

	// Inject system config
	userData["systemConfig"] = sysConfig
	// Inject username if missing (for consistency)
	if _, ok := userData["username"]; !ok {
		userData["username"] = username
	}

	c.JSON(http.StatusOK, userData)
}

func GetWidget(c *gin.Context) {
	username := c.GetString("username")
	if username == "" {
		username = "admin"
	}

	var sysConfig models.SystemConfig
	utils.ReadJSON(config.SystemConfigFile, &sysConfig)

	userFile := filepath.Join(config.UsersDir, username+".json")
	if username == "admin" && sysConfig.AuthMode == "single" {
		userFile = filepath.Join(config.DataDir, "data.json")
	}

	var userData map[string]interface{}
	if err := utils.ReadJSON(userFile, &userData); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User data not found"})
		return
	}

	widgets, ok := userData["widgets"].([]interface{})
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Widgets not found"})
		return
	}

	id := c.Param("id")
	for _, w := range widgets {
		if widgetMap, ok := w.(map[string]interface{}); ok {
			if wId, ok := widgetMap["id"].(string); ok && wId == id {
				data, _ := widgetMap["data"]
				c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
				return
			}
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Widget not found"})
}

func SaveData(c *gin.Context) {
	username := c.GetString("username")
	if username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 1. Bind to map to capture EVERYTHING sent by frontend
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	var sysConfig models.SystemConfig
	utils.ReadJSON(config.SystemConfigFile, &sysConfig)

	userFile := filepath.Join(config.UsersDir, username+".json")
	if username == "admin" && sysConfig.AuthMode == "single" {
		userFile = filepath.Join(config.DataDir, "data.json")
	}

	// 2. Read existing data to map to preserve EVERYTHING in file
	var existingData map[string]interface{}
	utils.ReadJSON(userFile, &existingData)
	if existingData == nil {
		existingData = make(map[string]interface{})
	}

	// 3. Handle Password Hashing
	// Check if payload has a password string
	if pwd, ok := payload["password"].(string); ok && pwd != "" {
		// Hash new password
		hashed, err := utils.HashPassword(pwd)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		payload["password"] = hashed
	} else {
		// Keep existing password
		if existingPwd, ok := existingData["password"]; ok {
			payload["password"] = existingPwd
		}
	}

	// 4. Merge other fields?
	// Actually, payload contains the full state of groups, widgets, appConfig etc.
	// So we can just use payload as the new state, but we should preserve top-level keys
	// that might be missing in payload but present in existingData (if any).
	// Frontend sends: groups, widgets, appConfig, rssFeeds, rssCategories.
	// If there are other top-level keys in existingData (like "created_at"?), we might want to keep them.
	for k, v := range existingData {
		if _, exists := payload[k]; !exists {
			payload[k] = v
		}
	}

	// Clean up legacy "items" field if "groups" is present in payload
	// This prevents the issue where deleting all groups causes legacy items to reappear as a "Default Group"
	if _, hasGroups := payload["groups"]; hasGroups {
		delete(payload, "items")
	}

	// Ensure username is set
	if _, ok := payload["username"]; !ok {
		payload["username"] = username
	}

	if err := utils.WriteJSON(userFile, payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ImportData handles importing JSON configuration
func ImportData(c *gin.Context) {
	// Re-use SaveData logic as it handles the exact same payload structure
	SaveData(c)
}

func SaveDefault(c *gin.Context) {
	username := c.GetString("username")
	// Only allow authenticated users (and maybe check for admin if needed, but for now just auth)
	if username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var sysConfig models.SystemConfig
	utils.ReadJSON(config.SystemConfigFile, &sysConfig)

	// Identify current user's file
	userFile := filepath.Join(config.UsersDir, username+".json")
	if username == "admin" && sysConfig.AuthMode == "single" {
		userFile = filepath.Join(config.DataDir, "data.json")
	}

	// Read current data
	var userData map[string]interface{}
	if err := utils.ReadJSON(userFile, &userData); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User data not found"})
		return
	}

	// Remove sensitive/user-specific data before saving as default
	delete(userData, "password")
	delete(userData, "username")
	delete(userData, "created_at")

	// Save to default.json
	if err := utils.WriteJSON(config.DefaultFile, userData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save default template"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func ResetData(c *gin.Context) {
	username := c.GetString("username")
	if username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Load default data
	var defaultData map[string]interface{}
	if err := utils.ReadJSON(config.DefaultFile, &defaultData); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Default template not found"})
		return
	}

	// Determine user file
	var sysConfig models.SystemConfig
	utils.ReadJSON(config.SystemConfigFile, &sysConfig)

	userFile := filepath.Join(config.UsersDir, username+".json")
	if username == "admin" && sysConfig.AuthMode == "single" {
		userFile = filepath.Join(config.DataDir, "data.json")
	}

	// Read current data to preserve password/username
	var currentData map[string]interface{}
	utils.ReadJSON(userFile, &currentData)

	// Merge: Use default data, but keep current password and username
	if currentData != nil {
		if pwd, ok := currentData["password"]; ok {
			defaultData["password"] = pwd
		}
		if usr, ok := currentData["username"]; ok {
			defaultData["username"] = usr
		}
	} else {
		// If current data is missing, ensure username is set
		defaultData["username"] = username
		// Password might be missing if it was empty
	}

	if err := utils.WriteJSON(userFile, defaultData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func GetSystemConfig(c *gin.Context) {
	var sysConfig models.SystemConfig
	utils.ReadJSON(config.SystemConfigFile, &sysConfig)
	c.JSON(http.StatusOK, sysConfig)
}

func UpdateSystemConfig(c *gin.Context) {
	username := c.GetString("username")
	if username != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	var sysConfig models.SystemConfig
	utils.ReadJSON(config.SystemConfigFile, &sysConfig)

	if v, ok := payload["authMode"].(string); ok {
		if v != "single" && v != "multi" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid authMode"})
			return
		}
		sysConfig.AuthMode = v
	}

	if v, ok := payload["enableDocker"].(bool); ok {
		sysConfig.EnableDocker = v
	}
	if v, ok := payload["dockerHost"].(string); ok {
		sysConfig.DockerHost = v
	}

	if err := utils.WriteJSON(config.SystemConfigFile, sysConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update system config"})
		return
	}

	c.JSON(http.StatusOK, sysConfig)
}
