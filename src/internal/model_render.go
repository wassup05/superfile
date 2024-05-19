package internal

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/lexers"
	"github.com/charmbracelet/lipgloss"
	"github.com/yorukot/ansichroma"
)

func sidebarRender(m model) string {
	s := sidebarTitleStyle.Render("     Super File")
	s += "\n"

	pinnedDivider := "\n" + sidebarTitleStyle.Render("󰐃 Pinned") + sidebarDividerStyle.Render(" ───────────") + "\n"
	disksDivider := "\n" + sidebarTitleStyle.Render("󱇰 Disks") + sidebarDividerStyle.Render(" ────────────") + "\n"

	totalHeight := 2
	for i := m.sidebarModel.renderIndex; i < len(m.sidebarModel.directories); i++ {
		if totalHeight >= m.mainPanelHeight {
			break
		} else {
			s += "\n"
		}

		directory := m.sidebarModel.directories[i]

		if directory.location == "Pinned+-*/=?" {
			s += pinnedDivider
			totalHeight += 3
			continue
		}

		if directory.location == "Disks+-*/=?" {
			if m.mainPanelHeight-totalHeight <= 2 {
				break
			}
			s += disksDivider
			totalHeight += 3
			continue
		}

		totalHeight++
		cursor := " "
		if m.sidebarModel.cursor == i && m.focusPanel == sidebarFocus {
			cursor = ""
		}

		if directory.location == m.fileModel.filePanels[m.filePanelFocusIndex].location {
			s += filePanelCursorStyle.Render(cursor) + sidebarSelectedStyle.Render(" "+truncateText(directory.name, sidebarWidth-2))
		} else {
			s += filePanelCursorStyle.Render(cursor) + sidebarStyle.Render(" "+truncateText(directory.name, sidebarWidth-2))
		}
	}

	return sideBarBorderStyle(m.mainPanelHeight, m.focusPanel).Render(s)
}

func filePanelRender(m model) string {
	// file panel
	f := make([]string, 10)
	for i, filePanel := range m.fileModel.filePanels {

		// check if cursor or render out of range
		if filePanel.cursor > len(filePanel.element)-1 {
			filePanel.cursor = 0
			filePanel.render = 0
		}
		m.fileModel.filePanels[i] = filePanel

		f[i] += filePanelTopDirectoryIconStyle.Render("   ") + filePanelTopPathStyle.Render(truncateTextBeginning(filePanel.location, m.fileModel.width-4)) + "\n"
		filePanelWidth := 0
		footerBorderWidth := 0

		if (m.fullWidth-sidebarWidth-(4+(len(m.fileModel.filePanels)-1)*2))%len(m.fileModel.filePanels) != 0 && i == len(m.fileModel.filePanels)-1 {
			if m.fileModel.filePreview.open {
				filePanelWidth = m.fileModel.width
				m.fileModel.filePreview.width = (m.fileModel.width + (m.fullWidth-sidebarWidth-(4+(len(m.fileModel.filePanels)-1)*2))%len(m.fileModel.filePanels))
			} else {
				filePanelWidth = (m.fileModel.width + (m.fullWidth-sidebarWidth-(4+(len(m.fileModel.filePanels)-1)*2))%len(m.fileModel.filePanels))
			}
			footerBorderWidth = m.fileModel.width + 7
		} else {
			filePanelWidth = m.fileModel.width
			footerBorderWidth = m.fileModel.width + 7
		}
		panelModeString := ""
		if filePanel.panelMode == browserMode {
			panelModeString = "󰈈 Browser"
		} else if filePanel.panelMode == selectMode {
			panelModeString = "󰆽 Select"
		}

		f[i] += filePanelDividerStyle(filePanel.focusType).Render(strings.Repeat(Config.BorderTop, filePanelWidth)) + "\n"
		f[i] += " " + filePanel.searchBar.View() + "\n"
		if len(filePanel.element) == 0 {
			f[i] += filePanelStyle.Render("   No such file or directory")
			bottomBorder := generateFooterBorder(fmt.Sprintf("%s%s%s", panelModeString, bottomMiddleBorderSplit, "0/0"), footerBorderWidth)
			f[i] = filePanelBorderStyle(m.mainPanelHeight, filePanelWidth, filePanel.focusType, bottomBorder).Render(f[i])
		} else {
			for h := filePanel.render; h < filePanel.render+panelElementHeight(m.mainPanelHeight) && h < len(filePanel.element); h++ {
				endl := "\n"
				if h == filePanel.render+panelElementHeight(m.mainPanelHeight)-1 || h == len(filePanel.element)-1 {
					endl = ""
				}
				cursor := " "
				// Check if the cursor needs to be displayed, if the user is using the search bar, the cursor is not displayed
				if h == filePanel.cursor && !filePanel.searchBar.Focused() {
					cursor = ""
				}
				isItemSelected := arrayContains(filePanel.selected, filePanel.element[h].location)
				if filePanel.renaming && h == filePanel.cursor {
					f[i] += filePanel.rename.View() + endl
				} else {
					f[i] += filePanelCursorStyle.Render(cursor+" ") + prettierName(filePanel.element[h].name, m.fileModel.width-5, filePanel.element[h].directory, isItemSelected, filePanelBGColor) + endl
				}
			}
			cursorPosition := strconv.Itoa(filePanel.cursor + 1)
			totalElement := strconv.Itoa(len(filePanel.element))

			bottomBorder := generateFooterBorder(fmt.Sprintf("%s%s%s/%s", panelModeString, bottomMiddleBorderSplit, cursorPosition, totalElement), footerBorderWidth)
			f[i] = filePanelBorderStyle(m.mainPanelHeight, filePanelWidth, filePanel.focusType, bottomBorder).Render(f[i])
		}
	}

	// file panel render together
	filePanelRender := ""
	for _, f := range f {
		filePanelRender = lipgloss.JoinHorizontal(lipgloss.Top, filePanelRender, f)
	}
	return filePanelRender
}

func processBarRender(m model) string {
	// save process in the array
	var processes []process
	for _, p := range m.processBarModel.process {
		processes = append(processes, p)
	}

	// sort by the process
	sort.Slice(processes, func(i, j int) bool {
		doneI := (processes[i].state == successful)
		doneJ := (processes[j].state == successful)

		// sort by done or not
		if doneI != doneJ {
			return !doneI
		}

		// if both not done
		if !doneI {
			completionI := float64(processes[i].done) / float64(processes[i].total)
			completionJ := float64(processes[j].done) / float64(processes[j].total)
			return completionI < completionJ // Those who finish first will be ranked later.
		}

		// if both done sort by the doneTime
		return processes[j].doneTime.Before(processes[i].doneTime)
	})

	// render
	processRender := ""
	renderTimes := 0

	for i := m.processBarModel.render; i < len(processes); i++ {
		if footerHeight < 14 && renderTimes == 2 {
			break
		}
		if renderTimes == 3 {
			break
		}
		process := processes[i]
		process.progress.Width = footerWidth(m.fullWidth) - 3
		symbol := ""
		cursor := ""
		if i == m.processBarModel.cursor {
			cursor = footerCursorStyle.Render("┃ ")
		} else {
			cursor = footerCursorStyle.Render("  ")
		}
		switch process.state {
		case failure:
			symbol = processErrorStyle.Render("")
		case successful:
			symbol = processSuccessfulStyle.Render("")
		case inOperation:
			symbol = processInOperationStyle.Render("󰥔")
		case cancel:
			symbol = processCancelStyle.Render("")
		}

		processRender += cursor + footerStyle.Render(truncateText(process.name, footerWidth(m.fullWidth)-7)+" ") + symbol + "\n"
		if renderTimes == 2 {
			processRender += cursor + process.progress.ViewAs(float64(process.done)/float64(process.total)) + ""
		} else if footerHeight < 14 && renderTimes == 1 {
			processRender += cursor + process.progress.ViewAs(float64(process.done)/float64(process.total))
		} else {
			processRender += cursor + process.progress.ViewAs(float64(process.done)/float64(process.total)) + "\n\n"
		}
		renderTimes++
	}

	if len(processes) == 0 {
		processRender += "\n   No processes running"
	}
	courseNumber := 0
	if len(m.processBarModel.processList) == 0 {
		courseNumber = 0
	} else {
		courseNumber = m.processBarModel.cursor + 1
	}
	bottomBorder := generateFooterBorder(fmt.Sprintf("%s/%s", strconv.Itoa(courseNumber), strconv.Itoa(len(m.processBarModel.processList))), footerWidth(m.fullWidth)-3)
	processRender = procsssBarBoarder(bottomElementHeight(footerHeight), footerWidth(m.fullWidth), bottomBorder, m.focusPanel).Render(processRender)

	return processRender
}

func metadataRender(m model) string {
	// process bar
	metaDataBar := ""
	if len(m.fileMetaData.metaData) == 0 && len(m.fileModel.filePanels[m.filePanelFocusIndex].element) > 0 && !m.fileModel.renaming {
		m.fileMetaData.metaData = append(m.fileMetaData.metaData, [2]string{"", ""})
		m.fileMetaData.metaData = append(m.fileMetaData.metaData, [2]string{" 󰥔  Loading metadata...", ""})
		go func() {
			m = returnMetaData(m)
		}()
	}
	maxKeyLength := 0
	sort.Slice(m.fileMetaData.metaData, func(i, j int) bool {
		comparisonFields := []string{"FileName", "FileSize", "FolderName", "FolderSize", "FileModifyDate", "FileAccessDate"}

		for _, field := range comparisonFields {
			if m.fileMetaData.metaData[i][0] == field {
				return true
			} else if m.fileMetaData.metaData[j][0] == field {
				return false
			}
		}

		// Default comparison
		return m.fileMetaData.metaData[i][0] < m.fileMetaData.metaData[j][0]
	})
	for _, data := range m.fileMetaData.metaData {
		if len(data[0]) > maxKeyLength {
			maxKeyLength = len(data[0])
		}
	}

	sprintfLength := maxKeyLength + 1
	valueLength := footerWidth(m.fullWidth) - maxKeyLength - 2
	if valueLength < footerWidth(m.fullWidth)/2 {
		valueLength = footerWidth(m.fullWidth)/2 - 2
		sprintfLength = valueLength
	}

	for i := m.fileMetaData.renderIndex; i < bottomElementHeight(footerHeight)+m.fileMetaData.renderIndex && i < len(m.fileMetaData.metaData); i++ {
		if i != m.fileMetaData.renderIndex {
			metaDataBar += "\n"
		}
		data := truncateMiddleText(m.fileMetaData.metaData[i][1], valueLength)
		metadataName := m.fileMetaData.metaData[i][0]
		if footerWidth(m.fullWidth)-maxKeyLength-3 < footerWidth(m.fullWidth)/2 {
			metadataName = truncateMiddleText(m.fileMetaData.metaData[i][0], valueLength)
		}
		metaDataBar += fmt.Sprintf("%-*s %s", sprintfLength, metadataName, data)

	}
	bottomBorder := generateFooterBorder(fmt.Sprintf("%s/%s", strconv.Itoa(m.fileMetaData.renderIndex+1), strconv.Itoa(len(m.fileMetaData.metaData))), footerWidth(m.fullWidth)-3)
	metaDataBar = metadataBoarder(bottomElementHeight(footerHeight), footerWidth(m.fullWidth), bottomBorder, m.focusPanel).Render(metaDataBar)

	return metaDataBar
}

func clipboardRender(m model) string {

	// render
	clipboardRender := ""
	if len(m.copyItems.items) == 0 {
		clipboardRender += "\n   No content in clipboard"
	} else {
		for i := 0; i < len(m.copyItems.items) && i < bottomElementHeight(footerHeight); i++ {
			if i == bottomElementHeight(footerHeight)-1 {
				clipboardRender += strconv.Itoa(len(m.copyItems.items)-i+1) + " item left...."
			} else {
				fileInfo, err := os.Stat(m.copyItems.items[i])
				if err != nil {
					outPutLog("Clipboard render function get item state error", err)
				}
				if !os.IsNotExist(err) {
					clipboardRender += clipboardPrettierName(m.copyItems.items[i], footerWidth(m.fullWidth)-3, fileInfo.IsDir(), false) + "\n"
				}
			}
		}
	}
	for i := 0; i < len(m.copyItems.items); i++ {

	}
	bottomWidth := 0

	if m.fullWidth%3 != 0 {
		bottomWidth = footerWidth(m.fullWidth + m.fullWidth%3 + 2)
	} else {
		bottomWidth = footerWidth(m.fullWidth)
	}
	clipboardRender = clipboardBoarder(bottomElementHeight(footerHeight), bottomWidth, Config.BorderBottom).Render(clipboardRender)

	return clipboardRender
}

func terminalSizeWarnRender(m model) string {
	fullWidthString := strconv.Itoa(m.fullWidth)
	fullHeightString := strconv.Itoa(m.fullHeight)
	minimumWidthString := strconv.Itoa(minimumWidth)
	minimumHeightString := strconv.Itoa(minimumHeight)
	if m.fullHeight < minimumHeight {
		fullHeightString = terminalTooSmall.Render(fullHeightString)
	}
	if m.fullWidth < minimumWidth {
		fullWidthString = terminalTooSmall.Render(fullWidthString)
	}
	fullHeightString = terminalCorrectSize.Render(fullHeightString)
	fullWidthString = terminalCorrectSize.Render(fullWidthString)

	heightString := mainStyle.Render(" Height = ")
	return fullScreenStyle(m.fullHeight, m.fullWidth).Render(`Terminal size too small:` + "\n" +
		"Width = " + fullWidthString +
		heightString + fullHeightString + "\n\n" +

		"Needed for current config:" + "\n" +
		"Width = " + terminalCorrectSize.Render(minimumWidthString) +
		heightString + terminalCorrectSize.Render(minimumHeightString))
}

func terminalSizeWarnAfterFirstRender(m model) string {
	minimumWidthInt := sidebarWidth + 20*len(m.fileModel.filePanels) + 20 - 1
	minimumWidthString := strconv.Itoa(minimumWidthInt)
	fullWidthString := strconv.Itoa(m.fullWidth)
	fullHeightString := strconv.Itoa(m.fullHeight)
	minimumHeightString := strconv.Itoa(minimumHeight)

	if m.fullHeight < minimumHeight {
		fullHeightString = terminalTooSmall.Render(fullHeightString)
	}
	if m.fullWidth < minimumWidthInt {
		fullWidthString = terminalTooSmall.Render(fullWidthString)
	}
	fullHeightString = terminalCorrectSize.Render(fullHeightString)
	fullWidthString = terminalCorrectSize.Render(fullWidthString)

	heightString := mainStyle.Render(" Height = ")
	return fullScreenStyle(m.fullHeight, m.fullWidth).Render(`You change your terminal size too small:` + "\n" +
		"Width = " + fullWidthString +
		heightString + fullHeightString + "\n\n" +

		"Needed for current config:" + "\n" +
		"Width = " + terminalCorrectSize.Render(minimumWidthString) +
		heightString + terminalCorrectSize.Render(minimumHeightString))
}

func typineModalRender(m model) string {
	previewPath := m.typingModal.location + "/" + m.typingModal.textInput.Value()

	fileLocation := filePanelTopDirectoryIconStyle.Render("   ") +
		filePanelTopPathStyle.Render(truncateTextBeginning(previewPath, modalWidth-4)) + "\n"

	confirm := modalConfirm.Render(" (" + hotkeys.ConfirmTyping[0] + ") Create ")
	cancel := modalCancel.Render(" (" + hotkeys.CancelTyping[0] + ") Cancel ")

	tip := confirm +
		lipgloss.NewStyle().Background(modalBGColor).Render("           ") +
		cancel

	return modalBorderStyle(modalHeight, modalWidth).Render(fileLocation + "\n" + m.typingModal.textInput.View() + "\n\n" + tip)
}

func introduceModalRender(m model) string {
	title := sidebarTitleStyle.Render(" Thanks for use superfile!!") + modalStyle.Render("\n You can read the following information before starting to use it!")
	vimUserWarn := processErrorStyle.Render("  ** Very importantly ** If you are a Vim/Nvim user, go to:\n  https://superfile.netlify.app/configure/custom-hotkeys/ to change your hotkey settings!")
	subOne := sidebarTitleStyle.Render("  (1)") + modalStyle.Render(" If this is your first time, make sure you read:\n      https://github.com/yorukot/superfile/wiki/Tutorial")
	subTwo := sidebarTitleStyle.Render("  (2)") + modalStyle.Render(" If you forget the relevant keys during use,\n      you can press \"?\" (shift+/) at any time to query the keys!")
	subThree := sidebarTitleStyle.Render("  (3)") + modalStyle.Render(" For more customization you can refer to:\n      https://github.com/yorukot/superfile/wiki")
	subFour := sidebarTitleStyle.Render("  (4)") + modalStyle.Render(" Thank you again for using superfile.\n      If you have any questions, please feel free to ask at:\n      https://github.com/yorukot/superfile\n      Of course, you can always open a new issue to share your idea \n      or report a bug!")
	return firstUseModal(m.helpMenu.height, m.helpMenu.width).Render(title + "\n\n" + vimUserWarn + "\n\n" + subOne + "\n\n" + subTwo + "\n\n" + subThree + "\n\n" + subFour + "\n\n")
}

func warnModalRender(m model) string {
	title := m.warnModal.title
	content := m.warnModal.content
	confirm := modalConfirm.Render(" (" + hotkeys.Confirm[0] + ") Confirm ")
	cancel := modalCancel.Render(" (" + hotkeys.Quit[0] + ") Cancel ")
	tip := confirm + lipgloss.NewStyle().Background(modalBGColor).Render("           ") + cancel
	return modalBorderStyle(modalHeight, modalWidth).Render(title + "\n\n" + content + "\n\n" + tip)
}

func helpMenuRender(m model) string {
	helpMenuContent := ""
	maxKeyLength := 0

	for _, data := range m.helpMenu.data {
		if data.subTitle == "" && len(data.hotkey[0]+data.hotkey[1])+3 > maxKeyLength {
			maxKeyLength = len(data.hotkey[0]+data.hotkey[1]) + 3
		}
	}

	valueLength := m.helpMenu.width - maxKeyLength - 2
	if valueLength < m.helpMenu.width/2 {
		valueLength = m.helpMenu.width/2 - 2
	}

	renderHotkeyLength := 0
	totalTitleCount := 0
	cursorBeenTitleCount := 0

	for i, data := range m.helpMenu.data {
		if data.subTitle != "" {
			if i < m.helpMenu.cursor {
				cursorBeenTitleCount++
			}
			totalTitleCount++
		}
	}

	for i := m.helpMenu.renderIndex; i < m.helpMenu.height+m.helpMenu.renderIndex && i < len(m.helpMenu.data); i++ {
		hotkey := ""

		if m.helpMenu.data[i].subTitle != "" {
			continue
		}

		if m.helpMenu.data[i].hotkey[1] == "" {
			hotkey = m.helpMenu.data[i].hotkey[0]
		} else {
			hotkey = fmt.Sprintf("%s | %s", m.helpMenu.data[i].hotkey[0], m.helpMenu.data[i].hotkey[1])
		}
		if len(helpMenuHotkeyStyle.Render(hotkey)) > renderHotkeyLength {
			renderHotkeyLength = len(helpMenuHotkeyStyle.Render(hotkey))
		}
	}

	for i := m.helpMenu.renderIndex; i < m.helpMenu.height+m.helpMenu.renderIndex && i < len(m.helpMenu.data); i++ {

		if i != m.helpMenu.renderIndex {
			helpMenuContent += "\n"
		}

		if m.helpMenu.data[i].subTitle != "" {
			helpMenuContent += helpMenuTitleStyle.Render(" " + m.helpMenu.data[i].subTitle)
			continue
		}

		hotkey := ""
		description := truncateText(m.helpMenu.data[i].description, valueLength)
		if m.helpMenu.data[i].hotkey[1] == "" {
			hotkey = m.helpMenu.data[i].hotkey[0]
		} else {
			hotkey = fmt.Sprintf("%s | %s", m.helpMenu.data[i].hotkey[0], m.helpMenu.data[i].hotkey[1])
		}
		cursor := "  "
		if m.helpMenu.cursor == i {
			cursor = filePanelCursorStyle.Render(" ")
		}
		helpMenuContent += cursor + modalStyle.Render(fmt.Sprintf("%*s%s", renderHotkeyLength, helpMenuHotkeyStyle.Render(hotkey+" "), modalStyle.Render(description)))
	}

	bottomBorder := generateFooterBorder(fmt.Sprintf("%s/%s", strconv.Itoa(m.helpMenu.cursor+1-cursorBeenTitleCount), strconv.Itoa(len(m.helpMenu.data)-totalTitleCount)), m.helpMenu.width-2)

	return helpMenuModalBorderStyle(m.helpMenu.height, m.helpMenu.width, bottomBorder).Render(helpMenuContent)
}

func filePreviewPanelRender(m model) string {
	previewLine := m.mainPanelHeight + 2

	box := filePreviewBox(previewLine, m.fileModel.filePreview.width)

	panel := m.fileModel.filePanels[m.filePanelFocusIndex]
	
	if len(panel.element) == 0 {
		return box.Render("\n ---  No content to preview ---")
	}

	fileInfo, err := os.Stat(panel.element[panel.cursor].location)
	if err != nil {
		outPutLog("error get file info", err)
		return box.Render("\n ---  Error get file info ---")
	}

	if fileInfo.IsDir() {
		return box.Render("\n ---  Unsupported formats ---")
	}

	format := lexers.Match(filepath.Base(panel.element[panel.cursor].location))
	if format != nil {
		codeHighlight, err := ansichroma.HighlightFromFile(panel.element[panel.cursor].location, previewLine, theme.CodeSyntaxHighlightTheme, theme.FilePanelBG)
		
		if err != nil {
			outPutLog("Error render code highlight", err)
			return box.Render("\n ---  Error render code highlight ---")
		}
		if codeHighlight == "" {
			return box.Render("\n --- empty ---")
		}
		codeHighlight = checkAndTruncateLineLengths(codeHighlight, m.fileModel.filePreview.width)
		return box.Render(codeHighlight)
	} else {
		textFile, err := isTextFile(panel.element[panel.cursor].location)
		if err != nil {
			outPutLog("Error check text file", err)
		}
		if textFile {
			var fileContent string 
			file, err := os.Open(panel.element[panel.cursor].location)
			if err != nil {
				outPutLog(err)
				return box.Render("\n ---  Error open file ---")
			}
			defer file.Close()
	
			scanner := bufio.NewScanner(file)
			lineCount := 0
	
			for scanner.Scan() {
				fileContent += scanner.Text() + "\n"
				lineCount++
				if previewLine > 0 && lineCount >= previewLine {
					break
				}
			}
	
			if err := scanner.Err(); err != nil {
				outPutLog(err)
				return box.Render("\n ---  Error open file ---")
			}

			textContent  := checkAndTruncateLineLengths(string(fileContent), m.fileModel.filePreview.width) 
			
			return box.Render(textContent)
		}
	}

	return box.Render("\n ---  Unsupported formats ---")
}