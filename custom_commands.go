package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/antonmedv/walk/overlay"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/charmbracelet/bubbles/key"
)

const (
	argTypeCurrentDir            int = 0
	argTypeCurrentFile           int = 1
	argTypeSelectedFiles         int = 2
	argTypeSelectedOrCurrentFile int = 3
	argTypeInput                 int = 4
)

type customCommand struct {
	description      string
	key              key.Binding
	cmd              string
	prompt           string
	completedMessage string
	argType          int
}

type askInputForCommandMsg struct{}
type textInputAcceptedMsg struct{}
type cmdMenuAcceptedMsg struct{}

type customCommands struct {
	commands      []customCommand
	keyCmdMenu    key.Binding
	menu          table.Model     // Menu of custom commands.
	textInput     textinput.Model // For asking text input from the user.
	textInputCmd  *customCommand  // A command that's waiting for the text input.
	statusMessage string
}

func (c *customCommands) init(config *appConfig) {
	// Init keybinding.
	c.keyCmdMenu = key.NewBinding(key.WithKeys("f9"))
	if config.Keys != nil {
		if config.Keys.CustomCommands != nil {
			c.keyCmdMenu = key.NewBinding(key.WithKeys(*config.Keys.CustomCommands))
		}
	}

	// Create the menu.
	menuRows := []table.Row{}

	if config.CustomCommands != nil {
		for _, cmd := range *config.CustomCommands {
			menuRow := []string{}
			menuRow = append(menuRow, cmd.Description)
			menuRow = append(menuRow, "") // key

			var customCommand customCommand
			customCommand.description = cmd.Description
			if len(cmd.Key) != 0 {
				menuRow[1] = cmd.Key
				customCommand.key = key.NewBinding(key.WithKeys(cmd.Key))
			}
			customCommand.cmd = cmd.Command
			customCommand.prompt = cmd.Prompt
			customCommand.completedMessage = cmd.CompletedMessage
			switch cmd.Args {
			case "currentDir":
				customCommand.argType = argTypeCurrentDir
			case "currentFile":
				customCommand.argType = argTypeCurrentFile
			case "selectedFiles":
				customCommand.argType = argTypeSelectedFiles
			case "selectedOrCurrentFile":
				customCommand.argType = argTypeSelectedOrCurrentFile
			case "input":
				customCommand.argType = argTypeInput
			default:
				log.Println("Invalid command args: ", cmd.Args)
			}

			c.commands = append(c.commands, customCommand)
			menuRows = append(menuRows, menuRow)
		}
	}

	menuColumns := []table.Column{
		{Title: "Command", Width: 20},
		{Title: "Key", Width: 20},
	}
	cmdMenu := table.New(
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
	cmdMenu.SetStyles(s)

	c.menu = cmdMenu

	// Create text input.
	ti := textinput.New()
	ti.CharLimit = 156
	ti.Width = 40
	c.textInput = ti
}

func (c *customCommands) executeCommand(m *model, command *customCommand, filePaths ...string) tea.Cmd {
	commandSlice := append(strings.Split(command.cmd, " "), filePaths...)
	execCmd := exec.Command(commandSlice[0], commandSlice[1:]...)
	return tea.ExecProcess(execCmd, func(err error) tea.Msg {
		if err != nil {
			c.statusMessage = fmt.Sprint("ERROR: ", err.Error())
		} else if len(command.completedMessage) > 0 {
			c.statusMessage = command.completedMessage
		}

		// Refresh the list. Files may have been created/deleted.
		m.list()

		return nil
	})
}

func (c *customCommands) executeCustomCommand(m *model, customCommand *customCommand) (tea.Cmd, bool) {
	if customCommand == nil {
		return nil, false
	}
	if customCommand.cmd == "" {
		return nil, false
	}

	switch customCommand.argType {
	case argTypeCurrentDir:
		{
			if fileEntry, ok := m.currentFile(); ok {
				return c.executeCommand(m, customCommand, fileEntry.dirPath), true
			} else if len(m.files) == 0 {
				return c.executeCommand(m, customCommand, m.path), true
			} else {
				return nil, true
			}
		}
	case argTypeCurrentFile:
		{
			currentFilePath, ok := m.filePath()
			if !ok {
				return nil, true
			}
			return c.executeCommand(m, customCommand, currentFilePath), true
		}
	case argTypeSelectedFiles:
		{
			selectedFilePaths := getSelectedFilePaths(m)
			if len(selectedFilePaths) == 0 {
				return nil, true
			}
			return c.executeCommand(m, customCommand, selectedFilePaths...), true
		}
	case argTypeSelectedOrCurrentFile:
		{
			selectedFilePaths := getSelectedFilePaths(m)
			if len(selectedFilePaths) == 0 {
				currentFilePath, ok := m.filePath()
				if !ok {
					return nil, true
				}
				selectedFilePaths = append(selectedFilePaths, currentFilePath)
			}
			return c.executeCommand(m, customCommand, selectedFilePaths...), true
		}
	case argTypeInput:
		{
			c.textInputCmd = customCommand
			return func() tea.Msg { return askInputForCommandMsg{} }, true
		}
	}
	log.Println("Invalid command arg type: ", customCommand.argType)
	return nil, false
}

func (c *customCommands) view(view string) string {
	dialogStyle := lipgloss.NewStyle().Border(lipgloss.NormalBorder())

	if c.textInput.Focused() {
		view = overlay.PlaceOverlay(5, 1, dialogStyle.Render(c.textInput.View()), view)
		//view += "\n" + m.extra.textInput.View()
	}

	if len(c.statusMessage) > 0 {
		view += "\n" + bar.Render(c.statusMessage)
	}

	if c.menu.Focused() {
		view = overlay.PlaceOverlay(5, 1, dialogStyle.Render(c.menu.View()), view)
	}

	return view
}

func (c *customCommands) updateCmdMenu(msg tea.Msg) (tea.Cmd, bool) {
	if !c.menu.Focused() {
		return nil, false
	}

	// Handle keyboard input.
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, keyEsc) {
			c.menu.Blur()
			return nil, true
		} else if key.Matches(msg, keyEnter) {
			c.menu.Blur()
			return func() tea.Msg { return cmdMenuAcceptedMsg{} }, true
		}
	}

	var cmd tea.Cmd
	c.menu, cmd = c.menu.Update(msg)
	return cmd, true
}

func (c *customCommands) updateTextInput(msg tea.Msg) (tea.Cmd, bool) {
	if !c.textInput.Focused() {
		return nil, false
	}

	// Handle keyboard input.
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, keyEsc) {
			c.textInput.Blur()
			return nil, true
		} else if key.Matches(msg, keyEnter) {
			c.textInput.Blur()
			return func() tea.Msg { return textInputAcceptedMsg{} }, true
		}
	}

	var cmd tea.Cmd
	c.textInput, cmd = c.textInput.Update(msg)
	return cmd, true
}

func (c *customCommands) update(m *model, msg tea.Msg) (tea.Cmd, bool) {
	if cmd, handled := c.updateTextInput(msg); handled {
		return cmd, handled
	}

	if cmd, handled := c.updateCmdMenu(msg); handled {
		return cmd, handled
	}

	switch msg := msg.(type) {
	case askInputForCommandMsg:
		if len(c.textInputCmd.prompt) > 0 {
			c.textInput.Prompt = c.textInputCmd.prompt
		} else {
			c.textInput.Prompt = "Enter input: "
		}
		c.textInput.SetValue("")
		c.textInput.Focus()
		return nil, true
	case textInputAcceptedMsg:
		{
			inputText := strings.TrimSpace(c.textInput.Value())
			if len(inputText) > 0 && c.textInputCmd != nil {
				if currentFile, ok := m.currentFile(); ok {
					return c.executeCommand(m, c.textInputCmd, currentFile.dirPath, inputText), true
				} else if len(m.files) == 0 {
					return c.executeCommand(m, c.textInputCmd, m.path, inputText), true
				}
			}
			return nil, true
		}
	case cmdMenuAcceptedMsg:
		{
			cmdIndex := c.menu.Cursor()
			if cmd, handled := c.executeCustomCommand(m, &c.commands[cmdIndex]); handled {
				return cmd, true
			}
		}
	case tea.KeyMsg:
		// Clear the status message when any key is pressed.
		c.statusMessage = ""

		if key.Matches(msg, c.keyCmdMenu) {
			c.menu.SetCursor(0)
			c.menu.Focus()
			return nil, true
		}

		for _, customCommand := range c.commands {
			if key.Matches(msg, customCommand.key) {
				if cmd, handled := c.executeCustomCommand(m, &customCommand); handled {
					return cmd, true
				}
			}
		}
	}

	return nil, false
}
