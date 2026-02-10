package config

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	BaseDir              string
	DataDir              string
	UsersDir             string
	SystemConfigFile     string
	DefaultFile          string
	SecretFile           string
	DocDir               string
	MusicDir             string
	BackgroundsDir       string
	MobileBackgroundsDir string
	IconCacheDir         string
	PublicDir            string
	ConfigVersionsDir    string
	SecretKey            []byte
)

func Init() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	// Adjust BaseDir if running from backend or frontend directory
	if filepath.Base(cwd) == "backend" || filepath.Base(cwd) == "frontend" {
		BaseDir = filepath.Dir(cwd)
	} else {
		BaseDir = cwd
	}

	DataDir = filepath.Join(BaseDir, "server", "data")
	UsersDir = filepath.Join(DataDir, "users")
	SystemConfigFile = filepath.Join(DataDir, "system.json")
	DefaultFile = filepath.Join(DataDir, "default.json")
	SecretFile = filepath.Join(DataDir, "secret.key")
	DocDir = filepath.Join(BaseDir, "server", "doc")
	MusicDir = filepath.Join(BaseDir, "server", "music")
	BackgroundsDir = filepath.Join(BaseDir, "server", "PC")
	MobileBackgroundsDir = filepath.Join(BaseDir, "server", "APP")
	IconCacheDir = filepath.Join(DataDir, "icon-cache")
	PublicDir = filepath.Join(BaseDir, "server", "public")
	ConfigVersionsDir = filepath.Join(DataDir, "config_versions")

	ensureDirs()
	loadSecretKey()
}

func ensureDirs() {
	dirs := []string{DataDir, UsersDir, DocDir, MusicDir, BackgroundsDir, MobileBackgroundsDir, IconCacheDir, PublicDir, ConfigVersionsDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Failed to create dir %s: %v", dir, err)
		}
	}
}

func loadSecretKey() {
	if _, err := os.Stat(SecretFile); err == nil {
		keyHex, err := os.ReadFile(SecretFile)
		if err == nil {
			trimmed := strings.TrimSpace(string(keyHex))
			if trimmed != "" {
				SecretKey = []byte(trimmed)
				return
			}
		}
	}
	if len(SecretKey) == 0 {
		bytes := make([]byte, 32)
		if _, err := rand.Read(bytes); err != nil {
			log.Fatal(err)
		}
		keyHex := hex.EncodeToString(bytes)
		if err := os.WriteFile(SecretFile, []byte(keyHex), 0600); err != nil {
			log.Fatal(err)
		}
		SecretKey = []byte(keyHex)
	}
}

func GetSecretKeyString() string {
    return string(SecretKey)
}
