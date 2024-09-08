package main

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
)

var TimeRanges = []string{"05", "15", "30", "60"}

type Config struct {
	AppHash     string              `json:"app_hash"`
	BotToken    string              `json:"bot_token"`
	Cameras     []CameraConfig      `json:"cameras"`
	Permissions []CameraPermissions `json:"permissions"`
	AppId       int32               `json:"app_id"`
	AdminId     int64               `json:"admin_id"`
}

func (c Config) String() string {
	return fmt.Sprintf("AppHash: %v\nAdminId: %v\nCameras: %v\nPermissions: %v", len(c.AppHash), c.AdminId, c.Cameras, c.Permissions)
}

func (c *Config) GetConfigPath() (string, error) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return path.Join(userHomeDir, ".config/cameras-bot"), nil
}

func (c *Config) GetTmpRecordingPath(userId int64, hashSeed string) (string, error) {
	configDir, err := c.GetConfigPath()
	if err != nil {
		return "", err
	}

	hash := hashify([]byte(hashSeed))

	fileName := fmt.Sprintf("%v_%v.mp4", userId, hash)

	return path.Join(configDir, fileName), nil
}

func (c *Config) GetSessionPath() (string, error) {
	configDir, err := c.GetConfigPath()
	if err != nil {
		return "", err
	}

	return path.Join(configDir, "session"), nil
}

func (c *Config) GetPermissionsFor(userId int64) *CameraPermissions {
	for _, permissions := range c.Permissions {
		if permissions.UserId == userId {
			return &permissions
		}
	}

	return nil
}

func (c *Config) Setup() error {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := path.Join(userHomeDir, ".config/cameras-bot")
	fileSystem := os.DirFS(configDir)
	jsonBytes, err := fs.ReadFile(fileSystem, "config.json")
	if err != nil {
		return err
	}

	err = json.Unmarshal(jsonBytes, &c)
	if err != nil {
		return err
	}

	return nil
}

type CameraConfig struct {
	Tag  string `json:"tag"`
	Name string `json:"name"`
	User string `json:"user"`
	Pass string `json:"pass"`
	Host string `json:"host"`
}

func (c CameraConfig) String() string {
	return fmt.Sprintf("{Name: %v, Tag: %v, Host: %v}", c.Name, c.Tag, c.Host)
}

func (c *CameraConfig) Image() string {
	url := url.URL{
		Scheme:   "http",
		Host:     c.Host,
		Path:     "ISAPI/Streaming/channels/101/picture",
		RawQuery: "snapShotImageType=JPEG",
	}

	return url.String()
}

func (c *CameraConfig) Stream() string {
	url := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%v:%v", c.Host, 554),
		User:   url.UserPassword(c.User, c.Pass),
		Path:   "ISAPI/Streaming/Channels/101",
	}

	return url.String()
}

type CameraPermissions struct {
	Tags   []string ``
	UserId int64    `json:"user_id"`
}

func (p CameraPermissions) String() string {
	return fmt.Sprintf("{UserId: %v, Tags: %v}", p.UserId, p.Tags)
}

func hashify(bytes []byte) string {
	hasher := sha1.New()
	hasher.Write(bytes)

	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}
