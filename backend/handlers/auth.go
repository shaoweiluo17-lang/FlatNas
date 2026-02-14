package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"flatnasgo-backend/config"
	"flatnasgo-backend/models"
	"flatnasgo-backend/utils"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var sysConfig models.SystemConfig
	utils.ReadJSON(config.SystemConfigFile, &sysConfig)

	if sysConfig.AuthMode == "single" && req.Username == "" {
		req.Username = "admin"
	}
	if req.Username == "" {
		req.Username = "admin"
	}

	userFile := filepath.Join(config.UsersDir, req.Username+".json")
	if req.Username == "admin" && sysConfig.AuthMode == "single" {
		// Single mode admin data is in data.json
		userFile = filepath.Join(config.DataDir, "data.json")
	}

	var user models.User
	match := false

	if err := utils.ReadJSON(userFile, &user); err != nil {
		// If admin user not found, create default admin
		if req.Username == "admin" {
			hashed, err := bcrypt.GenerateFromPassword([]byte("admin"), 10)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
				return
			}
			user = models.User{
				Username: "admin",
				Password: string(hashed),
			}
			// Ensure directory exists
			if err := utils.WriteJSON(userFile, user); err == nil {
				// Successfully created default admin, now check password
				err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
				if err == nil {
					match = true
				}
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create default user"})
				return
			}
		} else {
						c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在或密码错误"})
						return
					}	} else {
		// User exists logic...
		storedPwd := user.Password
		if storedPwd == "" {
			storedPwd = "admin"
		}

		if len(storedPwd) > 0 && storedPwd[0] == '$' {
			err := bcrypt.CompareHashAndPassword([]byte(storedPwd), []byte(req.Password))
			if err == nil {
				match = true
			}
		} else {
			if req.Password == storedPwd {
				match = true
				hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
					return
				}
				user.Password = string(hashed)
				if err := utils.WriteJSON(userFile, user); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save user"})
					return
				}
			}
		}
	}

	if match {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"username": req.Username,
			"exp":      time.Now().Add(time.Hour * 24 * 30).Unix(),
		})
		tokenString, err := token.SignedString([]byte(config.GetSecretKeyString()))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sign token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "token": tokenString, "username": req.Username})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Password incorrect"})
	}
}

type RegisterRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	InviteCode string `json:"inviteCode,omitempty"`
}

type AddUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LicenseRequest struct {
	Key string `json:"key"`
}

func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码不能为空"})
		return
	}

	if req.Username == "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能注册为管理员账号"})
		return
	}

	// Check if registration is allowed
	var sysConfig models.SystemConfig
	utils.ReadJSON(config.SystemConfigFile, &sysConfig)

	if !sysConfig.AllowRegistration && req.InviteCode == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "注册功能已关闭"})
		return
	}

	// If invite code is provided, verify it
	if req.InviteCode != "" {
		inviteCodesFile := filepath.Join(config.DataDir, "invite_codes.json")
		var inviteCodes []models.InviteCode
		utils.ReadJSON(inviteCodesFile, &inviteCodes)

		validCode := false
		var codeIndex = -1
		var invalidReason string

		for i, code := range inviteCodes {
			if code.Code == req.InviteCode && code.IsActive {
				// Check if expired
				if code.ExpiresAt > 0 && code.ExpiresAt < time.Now().Unix() {
					invalidReason = "邀请码已过期"
					continue
				}
				// Check if max uses reached
				if code.MaxUses > 0 && code.UsedCount >= code.MaxUses {
					invalidReason = "邀请码使用次数已达上限"
					continue
				}
				validCode = true
				codeIndex = i
				break
			}
		}

		if !validCode {
			if invalidReason != "" {
				c.JSON(http.StatusForbidden, gin.H{"error": invalidReason})
			} else {
				c.JSON(http.StatusForbidden, gin.H{"error": "无效的邀请码"})
			}
			return
		}

		// Update used count
		inviteCodes[codeIndex].UsedCount++
		utils.WriteJSON(inviteCodesFile, inviteCodes)
	}

	userFile := filepath.Join(config.UsersDir, req.Username+".json")
	if _, err := os.Stat(userFile); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	user := models.User{
		Username: req.Username,
		Password: string(hashed),
	}

	if err := utils.WriteJSON(userFile, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func GetUsers(c *gin.Context) {
	username := c.GetString("username")
	if username != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	files, err := os.ReadDir(config.UsersDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read users directory"})
		return
	}

	var users []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			name := strings.TrimSuffix(file.Name(), ".json")
			if name != "admin" {
				users = append(users, name)
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func AddUser(c *gin.Context) {
	var req AddUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码不能为空"})
		return
	}

	if req.Username == "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能手动添加管理员账号"})
		return
	}

	userFile := filepath.Join(config.UsersDir, req.Username+".json")
	if _, err := os.Stat(userFile); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	user := models.User{
		Username: req.Username,
		Password: string(hashed),
	}

	if err := utils.WriteJSON(userFile, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func DeleteUser(c *gin.Context) {
	currentUsername := c.GetString("username")
	if currentUsername != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	username := c.Param("usr")
	if username == "" || username == "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid username"})
		return
	}

	userFile := filepath.Join(config.UsersDir, username+".json")
	if err := os.Remove(userFile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func UploadLicense(c *gin.Context) {
	var req LicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	licenseFile := filepath.Join(config.DataDir, "license.key")
	if err := os.WriteFile(licenseFile, []byte(req.Key), 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save license"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

type GenerateInviteCodeRequest struct {
	MaxUses     int    `json:"maxUses"`     // 0 for unlimited
	ExpiresIn   int    `json:"expiresIn"`   // Days until expiration, 0 for never
	Description string `json:"description"`
}

func GetInviteCodes(c *gin.Context) {
	username := c.GetString("username")
	if username != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	inviteCodesFile := filepath.Join(config.DataDir, "invite_codes.json")
	var inviteCodes []models.InviteCode
	utils.ReadJSON(inviteCodesFile, &inviteCodes)

	c.JSON(http.StatusOK, gin.H{"success": true, "codes": inviteCodes})
}

func GenerateInviteCode(c *gin.Context) {
	username := c.GetString("username")
	if username != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	var req GenerateInviteCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Generate random code
	bytes := make([]byte, 8)
	rand.Read(bytes)
	code := hex.EncodeToString(bytes)

	var expiresAt int64 = 0
	if req.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(req.ExpiresIn) * 24 * time.Hour).Unix()
	}

	newCode := models.InviteCode{
		Code:        code,
		CreatedBy:   username,
		CreatedAt:   time.Now().Unix(),
		MaxUses:     req.MaxUses,
		UsedCount:   0,
		ExpiresAt:   expiresAt,
		IsActive:    true,
		Description: req.Description,
	}

	inviteCodesFile := filepath.Join(config.DataDir, "invite_codes.json")
	var inviteCodes []models.InviteCode
	utils.ReadJSON(inviteCodesFile, &inviteCodes)
	inviteCodes = append(inviteCodes, newCode)

	if err := utils.WriteJSON(inviteCodesFile, inviteCodes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save invite code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "code": newCode})
}

func DeleteInviteCode(c *gin.Context) {
	username := c.GetString("username")
	if username != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Code required"})
		return
	}

	inviteCodesFile := filepath.Join(config.DataDir, "invite_codes.json")
	var inviteCodes []models.InviteCode
	utils.ReadJSON(inviteCodesFile, &inviteCodes)

	found := false
	for i, inviteCode := range inviteCodes {
		if inviteCode.Code == code {
			inviteCodes = append(inviteCodes[:i], inviteCodes[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invite code not found"})
		return
	}

	if err := utils.WriteJSON(inviteCodesFile, inviteCodes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete invite code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
