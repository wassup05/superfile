package internal

import (
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	variable "github.com/yorukot/superfile/src/config"
)

// Change file panel mode (select mode or browser mode)
func (m *model) changeFilePanelMode() {
	panel := &m.fileModel.filePanels[m.filePanelFocusIndex]
	if panel.panelMode == selectMode {
		panel.selected = panel.selected[:0]
		panel.panelMode = browserMode
	} else if panel.panelMode == browserMode {
		panel.panelMode = selectMode
	}
}

// Back to parent directory
func (m *model) parentDirectory() {
	panel := &m.fileModel.filePanels[m.filePanelFocusIndex]
	panel.directoryRecords[panel.location] = directoryRecord{
		directoryCursor: panel.cursor,
		directoryRender: panel.render,
	}
	fullPath := panel.location
	parentDir := filepath.Dir(fullPath)
	panel.location = parentDir
	curDirectoryRecord, hasRecord := panel.directoryRecords[panel.location]
	if hasRecord {
		panel.cursor = curDirectoryRecord.directoryCursor
		panel.render = curDirectoryRecord.directoryRender
	} else {
		panel.cursor = 0
		panel.render = 0
	}
}

// Enter directory or open file with default application
func (m *model) enterPanel() {
	panel := &m.fileModel.filePanels[m.filePanelFocusIndex]

	if len(panel.element) == 0 {
		return
	}

	if panel.element[panel.cursor].directory {
		panel.directoryRecords[panel.location] = directoryRecord{
			directoryCursor: panel.cursor,
			directoryRender: panel.render,
		}
		panel.location = panel.element[panel.cursor].location
		curDirectoryRecord, hasRecord := panel.directoryRecords[panel.location]
		if hasRecord {
			panel.cursor = curDirectoryRecord.directoryCursor
			panel.render = curDirectoryRecord.directoryRender
		} else {
			panel.cursor = 0
			panel.render = 0
		}
		panel.searchBar.SetValue("")
	} else if !panel.element[panel.cursor].directory {
		fileInfo, err := os.Lstat(panel.element[panel.cursor].location)
		if err != nil {
			slog.Error("Error while getting file info", "error", err)
			return
		}

		if fileInfo.Mode()&os.ModeSymlink != 0 {
			targetPath, symlink_err := filepath.EvalSymlinks(panel.element[panel.cursor].location)
			if symlink_err != nil {
				return
			}

			targetInfo, lstat_err := os.Lstat(targetPath)

			if lstat_err != nil {
				return
			}

			if targetInfo.IsDir() {
				m.fileModel.filePanels[m.filePanelFocusIndex].location = targetPath
			}

			return
		}

		openCommand := "xdg-open"
		if runtime.GOOS == "darwin" {
			openCommand = "open"
		} else if runtime.GOOS == "windows" {

			dllpath := filepath.Join(os.Getenv("SYSTEMROOT"), "System32", "rundll32.exe")
			dllfile := "url.dll,FileProtocolHandler"

			cmd := exec.Command(dllpath, dllfile, panel.element[panel.cursor].location)
			err = cmd.Start()
			if err != nil {
				slog.Error("Error while open file with", "error", err)
			}

			return
		}

		cmd := exec.Command(openCommand, panel.element[panel.cursor].location)
		err = cmd.Start()
		if err != nil {
			slog.Error("Error while open file with", "error", err)
		}

	}

}

// Switch to the directory where the sidebar cursor is located
func (m *model) sidebarSelectDirectory() {
	// We can't do this when we have only divider directories
	// m.sidebarModel.directories[m.sidebarModel.cursor].location would point to a divider dir.
	if m.sidebarModel.noActualDir() {
		return
	}
	m.focusPanel = nonePanelFocus
	panel := &m.fileModel.filePanels[m.filePanelFocusIndex]

	panel.directoryRecords[panel.location] = directoryRecord{
		directoryCursor: panel.cursor,
		directoryRender: panel.render,
	}

	panel.location = m.sidebarModel.directories[m.sidebarModel.cursor].location
	curDirectoryRecord, hasRecord := panel.directoryRecords[panel.location]
	if hasRecord {
		panel.cursor = curDirectoryRecord.directoryCursor
		panel.render = curDirectoryRecord.directoryRender
	} else {
		panel.cursor = 0
		panel.render = 0
	}
	panel.focusType = focus
}

// Select all item in the file panel (only work on select mode)
func (m *model) selectAllItem() {
	panel := &m.fileModel.filePanels[m.filePanelFocusIndex]
	for _, item := range panel.element {
		panel.selected = append(panel.selected, item.location)
	}
}

// Select the item where cursor located (only work on select mode)
func (panel *filePanel) singleItemSelect() {

	if len(panel.element) > 0 && panel.cursor >= 0 && panel.cursor < len(panel.element) {
		elementLocation := panel.element[panel.cursor].location

		if arrayContains(panel.selected, elementLocation) {
			// This is inefficient. Once you select 1000 items,
			// each select / deselect operation can take 1000 operations
			// It can be easily made constant time.
			// Todo : (performance)convert panel.selected to a set (map[string]struct{})
			panel.selected = removeElementByValue(panel.selected, elementLocation)
		} else {
			panel.selected = append(panel.selected, elementLocation)
		}
	}
}

// Toggle dotfile display or not
func (m *model) toggleDotFileController() {
	newToggleDotFile := ""
	if m.toggleDotFile {
		newToggleDotFile = "false"
		m.toggleDotFile = false
	} else {
		newToggleDotFile = "true"
		m.toggleDotFile = true
	}
	m.updatedToggleDotFile = true
	err := os.WriteFile(variable.ToggleDotFile, []byte(newToggleDotFile), 0644)
	if err != nil {
		slog.Error("Error while pinned folder function updatedData superfile data", "error", err)
	}

}

// Toggle dotfile display or not
func (m *model) toggleFooterController() {
	newToggleFooterFile := ""
	if m.toggleFooter {
		newToggleFooterFile = "false"
		m.toggleFooter = false
	} else {
		newToggleFooterFile = "true"
		m.toggleFooter = true
	}
	err := os.WriteFile(variable.ToggleFooter, []byte(newToggleFooterFile), 0644)
	if err != nil {
		slog.Error("Error while Toggle footer function updatedData superfile data", "error", err)
	}
	m.setHeightValues(m.fullHeight)

}

// Focus on search bar
func (m *model) searchBarFocus() {
	panel := &m.fileModel.filePanels[m.filePanelFocusIndex]
	if panel.searchBar.Focused() {
		panel.searchBar.Blur()
	} else {
		panel.searchBar.Focus()
		m.firstTextInput = true
	}

	// config search bar width
	panel.searchBar.Width = m.fileModel.width - 4
}

func (m *model) sidebarSearchBarFocus() {

	if m.sidebarModel.searchBar.Focused() {
		// Ideally Code should never reach here. Once sidebar is focussed, we should
		// not cause sidebarSearchBarFocus() event by pressing search key
		// Should we use Runtime panic asserts ?
		slog.Error("sidebarSearchBarFocus() called on Focussed sidebar")
		m.sidebarModel.searchBar.Blur()
	} else {
		m.sidebarModel.searchBar.Focus()
		m.firstTextInput = true
	}
}
