package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"path"
	"sort"

	"github.com/antonmedv/walk/overlay"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/charmbracelet/bubbles/key"
)

type dirHotlistItem struct {
	DirPath string `json:"path"`
}

type dirHotlist struct {
	keyDirHotlist key.Binding

	items []dirHotlistItem
	menu  table.Model
}

func (d *dirHotlist) init(config *appConfig) {
	d.keyDirHotlist = key.NewBinding(key.WithKeys("f9"))
	if config.Keys != nil {
		if config.Keys.DirHotlist != nil {
			d.keyDirHotlist = key.NewBinding(key.WithKeys(*config.Keys.DirHotlist))
		}
	}
}

func (d *dirHotlist) recreateMenu() {

	d.readFromJson()

	// Just sort alphabetically for now.
	sort.Slice(d.items, func(i, j int) bool {
		a := d.items[i]
		b := d.items[j]
		return a.DirPath < b.DirPath
	})

	menuRows := []table.Row{}

	for _, item := range d.items {
		menuRow := []string{}
		menuRow = append(menuRow, item.DirPath)
		menuRows = append(menuRows, menuRow)
	}

	menuColumns := []table.Column{
		{Title: "Path", Width: 50},
	}
	menu := table.New(
		table.WithColumns(menuColumns),
		table.WithRows(menuRows),
		table.WithHeight(7),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		Bold(false)
	s.Selected = cursor
	menu.SetStyles(s)

	d.menu = menu
}

func (d *dirHotlist) readFromJson() {
	d.items = []dirHotlistItem{}

	homeDir, ok := os.LookupEnv("HOME")
	if !ok {
		log.Println("HOME env var is not defined.")
		return
	}

	jsonPath := path.Join(homeDir, ".config", "walk_dirhotlist.json")

	if err := ensureDirExists(path.Dir(jsonPath)); err != nil {
		log.Println("Cannot create directory for ", jsonPath)
		return
	}

	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		return
	}

	jsonFile, err := os.Open(jsonPath)
	if err != nil {
		log.Println("Cannot read file: ", jsonPath)
		return
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	err = json.Unmarshal(byteValue, &d.items)
	if err != nil {
		log.Printf("Cannot parse %s: %s", jsonPath, err)
	}
}

func (d *dirHotlist) writeToJson() {
	homeDir, ok := os.LookupEnv("HOME")
	if !ok {
		log.Println("HOME env var is not defined.")
		return
	}

	jsonPath := path.Join(homeDir, ".config", "walk_dirhotlist.json")

	if err := ensureDirExists(path.Dir(jsonPath)); err != nil {
		log.Println("Cannot create directory for ", jsonPath)
		return
	}

	jsonFile, err := os.Create(jsonPath)
	if err != nil {
		log.Println("Cannot create file: ", jsonPath)
		return
	}
	defer jsonFile.Close()

	bytes, err := json.MarshalIndent(&d.items, "", "  ")
	if err != nil {
		log.Printf("Cannot marshal dir hotlist to JSON: %s", err)
		return
	}

	_, err = jsonFile.Write(bytes)
	if err != nil {
		log.Printf("Cannot write %s: %s", jsonPath, err)
		return
	}
}

func (d *dirHotlist) update(m *model, msg tea.Msg) (tea.Cmd, bool) {
	if d.menu.Focused() {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if key.Matches(msg, keyEsc) {
				d.menu.Blur()
				return nil, true
			} else if key.Matches(msg, keyEnter) {
				d.menu.Blur()
				itemIndex := d.menu.Cursor()
				if 0 <= itemIndex && itemIndex < len(d.items) {
					enterDirectory(m, d.items[itemIndex].DirPath)
				}
				return nil, true
			} else if key.Matches(msg, keyA) {
				if currentFile, ok := m.currentFile(); ok {
					d.items = append(d.items, dirHotlistItem{DirPath: currentFile.dirPath})
				} else {
					d.items = append(d.items, dirHotlistItem{DirPath: m.path})
				}
				d.writeToJson()
				d.recreateMenu()
				d.menu.Focus()
			} else if key.Matches(msg, keyD) {
				itemIndex := d.menu.Cursor()
				if 0 <= itemIndex && itemIndex < len(d.items) {
					d.items = append(d.items[0:itemIndex], d.items[itemIndex+1:]...)
					d.writeToJson()
					d.recreateMenu()
					d.menu.Focus()
				}
			}
		}

		var cmd tea.Cmd
		d.menu, cmd = d.menu.Update(msg)
		return cmd, true

	} else {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if key.Matches(msg, d.keyDirHotlist) {
				d.recreateMenu()
				d.menu.SetCursor(0)
				d.menu.Focus()
				return nil, true
			}
		}
	}

	return nil, false
}

func (d *dirHotlist) view(view string) string {
	if !d.menu.Focused() {
		return view
	}
	dialogStyle := lipgloss.NewStyle().Border(lipgloss.NormalBorder())
	return overlay.PlaceOverlay(5, 1, dialogStyle.Render(d.menu.View()), view)
}
