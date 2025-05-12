package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"path"
)

type appConfig struct {
	Keys           *keysConfig            `json:"keys,omitempty"`
	Colors         *colorsConfig          `json:"colors,omitempty"`
	Layout         *layoutConfig          `json:"layout,omitempty"`
	Editor         *string                `json:"editor,omitempty"`
	CustomCommands *[]customCommandConfig `json:"customCommands,omitempty"`
}

type keysConfig struct {
	ForceQuit      *string `json:"forceQuit,omitempty"`
	Quit           *string `json:"quit,omitempty"`
	QuitQ          *string `json:"quitQ,omitempty"`
	UpDir          *string `json:"upDir,omitempty"`
	OpenDir        *string `json:"openDir,omitempty"`
	OpenTree       *string `json:"openTree,omitempty"`
	CloseTree      *string `json:"closeTree,omitempty"`
	Back           *string `json:"back,omitempty"`
	Select         *string `json:"select,omitempty"`
	CustomCommands *string `json:"customCommands,omitempty"`
	DirHotlist     *string `json:"dirHotlist,omitempty"`
}

type colorsConfig struct {
	Main         *colorConfig `json:"main,omitempty"`
	Cursor       *colorConfig `json:"cursor,omitempty"`
	StatusBar    *colorConfig `json:"statusBar,omitempty"`
	Directory    *colorConfig `json:"directory,omitempty"`
	Symlink      *colorConfig `json:"symlink,omitempty"`
	Executable   *colorConfig `json:"executable,omitempty"`
	SelectedFile *colorConfig `json:"selectedFile,omitempty"`
}

type layoutConfig struct {
	StatusBar       *string `json:"statusBar,omitempty"`
	FileInfo        *string `json:"fileInfo,omitempty"`
	MaxColumns      *int    `json:"maxColumns,omitempty"`
	LongListColumns *int    `json:"longListColumns,omitempty"`
	LongListLimit   *int    `json:"longListLimit,omitempty"`
	ColumnSeparator *string `json:"columnSeparator,omitempty"`
	SelectionMark   *string `json:"selectionMark,omitempty"`
}

type colorConfig struct {
	Foreground *string `json:"fg,omitempty"`
	Background *string `json:"bg,omitempty"`
}

type customCommandConfig struct {
	Description      string `json:"description"`
	Key              string `json:"key,omitempty"`
	Command          string `json:"cmd"`
	Prompt           string `json:"prompt,omitempty"`
	CompletedMessage string `json:"completedMessage,omitempty"`
	Args             string `json:"args"`
}

func readConfig() appConfig {
	var config appConfig

	homeDir, ok := os.LookupEnv("HOME")
	if !ok {
		log.Println("HOME env var is not defined.")
		return config
	}

	jsonPath := path.Join(homeDir, ".config", "walk.json")

	if err := ensureDirExists(path.Dir(jsonPath)); err != nil {
		log.Println("Cannot create directory for ", jsonPath)
		return config
	}

	jsonFile, err := os.Open(jsonPath)
	if err != nil {
		log.Println("Cannot read config file: ", jsonPath)
		return config
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	err = json.Unmarshal(byteValue, &config)
	if err != nil {
		log.Printf("Cannot parse %s: %s", jsonPath, err)
	}

	return config
}
