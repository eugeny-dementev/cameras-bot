package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/url"
	"os"
	"path"
	"regexp"
)

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
	Tag    string `json:"tag"`
	Name   string `json:"name"`
	Stream string `json:"stream"`
	Image  string `json:"image"`
}

func (c CameraConfig) String() string {
	parsedUrl, err := url.Parse(c.Stream)
	if err != nil {
		log.Panic("cannot parse provided input URL", err)
	}

	re := regexp.MustCompile("[a-f0-9]")

	parsedUrl.User = url.UserPassword("root", "root")
	parsedUrl.Host = re.ReplaceAllString(parsedUrl.Host, "*")

	return fmt.Sprintf("{Name: %v, Tag: %v, URL: %v}", c.Name, c.Tag, parsedUrl)
}

type CameraPermissions struct {
	Tags   []string `json:"tags"`
	UserId int64    `json:"user_id"`
}

func (p CameraPermissions) String() string {
	return fmt.Sprintf("{UserId: %v, Tags: %v}", p.UserId, p.Tags)
}
