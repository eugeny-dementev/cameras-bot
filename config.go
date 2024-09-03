package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
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
	for _, perm := range c.Permissions {
		if perm.UserId == userId {
			return &perm
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

func getConfig() Config {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	configDir := path.Join(userHomeDir, ".config/cameras-bot")
	fileSystem := os.DirFS(configDir)
	jsonBytes, err := fs.ReadFile(fileSystem, "config.json")
	if err != nil {
		log.Fatal(err)
	}

	conf := Config{}
	err = json.Unmarshal(jsonBytes, &conf)
	if err != nil {
		log.Fatal(err)
	}

	return conf
}
