package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"log"
	"strconv"
)

type forcedVariant struct {
	fyne.Theme
	variant fyne.ThemeVariant
}

func (f *forcedVariant) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return f.Theme.Color(name, f.variant)
}

func changeTheme(checked bool) {
	if checked {
		myWin.App.Settings().SetTheme(&forcedVariant{Theme: theme.DefaultTheme(), variant: theme.VariantDark})
	} else {
		myWin.App.Settings().SetTheme(&forcedVariant{Theme: theme.DefaultTheme(), variant: theme.VariantLight})
	}
}

func (app *Config) makeUI() {

	flashEdges = []FlashEdge{}

	//changeTheme(true) // Start with black theme

	app.statusLine = makeStatusLine(app)

	app.textOut = []string{}

	topItem := container.NewVBox(app.statusLine)

	blackThemeCheckbox := widget.NewCheck("Dark theme", func(checked bool) { changeTheme(checked) })
	//leftItem.Add(blackThemeCheckbox)
	blackThemeCheckbox.SetChecked(true)

	// Compose the left hand column element of the main Border layout
	leftItem := container.NewVBox()
	helpButton := widget.NewButton("Help", func() { showHelp() })
	leftItem.Add(helpButton)

	app.curBaudRate = baudrate

	leftItem.Add(canvas.NewText("Serial ports available", nil))
	app.selectComPort = widget.NewSelect([]string{}, func(value string) { handleComPortSelection(value) })
	leftItem.Add(app.selectComPort)
	app.comPortInUse = widget.NewLabel("Serial port open: none")
	leftItem.Add(app.comPortInUse)

	closePortButton := widget.NewButton("Close serial port", func() { closeCurrentPort() })
	leftItem.Add(closePortButton)

	leftItem.Add(widget.NewButton("Show 1pps history", func() { show1ppsHistory() }))

	leftItem.Add(blackThemeCheckbox)

	leftItem.Add(layout.NewSpacer())

	leftItem.Add(canvas.NewText("=========================", color.NRGBA{R: 180, A: 255}))

	//app.gpsUtcOffsetInUse = widget.NewLabel("GPS-UTC offset in use: TBD")
	//app.gpsUtcOffsetInUse = widget.NewLabel("")
	app.gpsUtcOffsetInUse = canvas.NewText("", nil)
	leftItem.Add(app.gpsUtcOffsetInUse)

	leftItem.Add(canvas.NewText("UTC event date/time", nil))
	app.utcEventTime = widget.NewEntry()
	app.utcEventTime.SetText(myWin.App.Preferences().StringWithFallback("UTCstartTime", ""))
	app.utcEventTime.OnSubmitted = func(stuff string) { processUTCeventTimeEntry(stuff) }
	leftItem.Add(app.utcEventTime)

	leftItem.Add(canvas.NewText("Recording length (sec)", nil))
	app.recordingLength = widget.NewEntry()
	app.recordingLength.SetText(myWin.App.Preferences().StringWithFallback("RecordingTime", ""))
	app.recordingLength.OnSubmitted = func(stuff string) { processRecordingLengthEntry(stuff) }
	leftItem.Add(app.recordingLength)

	myWin.autoRunFitsReaderCheckBox = widget.NewCheck("auto-run FitsReader", autoRunFitsReader)

	leftItem.Add(myWin.autoRunFitsReaderCheckBox)

	myWin.shutdownCheckBox = widget.NewCheck("shutdown CPU at end-of-recording", shutdownEnable)
	shutdownChecked := myWin.App.Preferences().BoolWithFallback("ShutdownComputerAtEndOfRecording", false)
	myWin.shutdownCheckBox.SetChecked(shutdownChecked)
	leftItem.Add(myWin.shutdownCheckBox)

	autoRunChecked := myWin.App.Preferences().BoolWithFallback("AutoRunFitsReader", true)
	myWin.autoRunFitsReaderCheckBox.SetChecked(autoRunChecked)

	app.armUTCbutton = widget.NewButton("Arm UTC start", func() { armUTCstart() })
	leftItem.Add(app.armUTCbutton)

	leftItem.Add(canvas.NewText("=========================", color.NRGBA{R: 180, A: 255}))

	leftItem.Add(layout.NewSpacer())

	leftItem.Add(widget.NewButton("Clear output", func() { clearSerialOutputDisplay() }))
	app.autoScroll = widget.NewCheck("Auto-scroll enabled", func(bool) {})
	app.autoScroll.SetChecked(true)
	leftItem.Add(app.autoScroll)

	column1 := container.NewVBox()

	column1.Add(layout.NewSpacer())

	column1.Add(widget.NewLabel("Display items"))

	app.gpggaCheckBox = widget.NewCheck("$GPGGA", func(bool) {})
	app.gpggaCheckBox.SetChecked(false)
	column1.Add(app.gpggaCheckBox)

	app.gprmcCheckBox = widget.NewCheck("$GPRMC", func(bool) {})
	app.gprmcCheckBox.SetChecked(false)
	column1.Add(app.gprmcCheckBox)

	app.gpdtmCheckBox = widget.NewCheck("$GPDTM", func(bool) {})
	app.gpdtmCheckBox.SetChecked(false)
	column1.Add(app.gpdtmCheckBox)

	app.pubxCheckBox = widget.NewCheck("$PUBX", func(bool) {})
	app.pubxCheckBox.SetChecked(true)
	column1.Add(app.pubxCheckBox)

	app.pCheckBox = widget.NewCheck("P", func(bool) {})
	app.pCheckBox.SetChecked(false)
	column1.Add(app.pCheckBox)

	app.modeCheckBox = widget.NewCheck("MODE", func(bool) {})
	app.modeCheckBox.SetChecked(false)
	column1.Add(app.modeCheckBox)

	column1.Add(layout.NewSpacer())

	flashIntensitySlider := widget.NewSlider(0, 3*255)

	sticky := myWin.App.Preferences().StringWithFallback("LedIntensity", "400")
	//fmt.Println("LedIntensity", sticky)
	ledSetting, _ := strconv.Atoi(sticky)
	flashIntensitySlider.Value = float64(ledSetting)

	flashIntensitySlider.Orientation = 1 // Vertical
	flashIntensitySlider.OnChangeEnded = func(value float64) {
		processFlashIntensitySliderChange(value)
	}
	flashIntensitySlider.Hidden = true

	myWin.flashIntensitySlider = flashIntensitySlider

	ledOnCheckBox := widget.NewCheck("LED on", func(clicked bool) { showIntensitySlider(clicked) })
	ledOnCheckBox.SetChecked(false)
	myWin.ledOnCheckbox = ledOnCheckBox
	column2 := container.NewVBox()
	column2.Add(layout.NewSpacer())
	column2.Add(ledOnCheckBox)
	rightItem := container.NewHBox(column1, column2, flashIntensitySlider)

	// Compose bottom element of the main Border layout

	app.cmdEntry = widget.NewEntry()
	app.cmdEntry.OnSubmitted = func(str string) { sendCommandToArduino("") }

	app.pathEntry = widget.NewEntry()

	bottomEntry := container.NewBorder( // top, bottom, left, right, center
		nil,
		nil,
		//bob,
		canvas.NewText("Enter IOTA GFT command:", nil),
		widget.NewButton("Help: commands", func() { showCommandHelp() }),
		app.cmdEntry,
	)
	app.cmdEntry.SetPlaceHolder("commands to be sent to the GFT go here - Press Enter to send")

	bottomItem := container.NewVBox(widget.NewSeparator(), bottomEntry)

	app.textOut = getInitialText()
	app.textOutDisplay = widget.NewList(
		func() int {
			return len(app.textOut)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(app.textOut[i])
		})

	// We will build the GUI around a Border container.

	content := container.NewBorder(
		topItem,
		bottomItem,
		leftItem,
		rightItem,
		app.textOutDisplay)

	app.MainWindow.SetContent(content)
}

func autoRunFitsReader(checked bool) {
	if myWin.shutdownCheckBox.Checked {
		checked = false
	}
	myWin.App.Preferences().SetBool("AutoRunFitsReader", checked)
	myWin.autoRunFitsReaderCheckBox.SetChecked(checked)
}

func shutdownEnable(checked bool) {
	myWin.App.Preferences().SetBool("ShutdownComputerAtEndOfRecording", checked)
	if checked {
		autoRunFitsReader(false)
		myWin.autoRunFitsReaderCheckBox.SetChecked(false)
	}
}

func processUTCeventTimeEntry(stuff string) {
	myWin.App.Preferences().SetString("UTCstartTime", stuff)
	log.Println(stuff)
}

func processRecordingLengthEntry(stuff string) {
	myWin.App.Preferences().SetString("RecordingTime", stuff)
	log.Println("Recording time set to: ", stuff)
}

//func changeLogAndEdgeFiles(path string) {
//	fmt.Println("New path given:", path)
//	if path != "" {
//		createLogAndFlashEdgeFiles("", "")
//		//deleteLogfile()
//		//deleteEdgeTimesFile()
//	}
//}

func showIntensitySlider(clicked bool) {
	myWin.flashIntensitySlider.Hidden = !clicked
	if clicked {
		processFlashIntensitySliderChange(myWin.flashIntensitySlider.Value)
		sendCommandToArduino("led on")
	} else {
		sendCommandToArduino("led off")
	}
}

func processFlashIntensitySliderChange(value float64) {
	v := int64(value)
	ledRange := v / 256
	level := v - ledRange*255
	levelStr := fmt.Sprintf("%0.0f", value)
	//fmt.Println("Saving LedIntensity as:", levelStr)
	myWin.App.Preferences().SetString("LedIntensity", levelStr)

	//fmt.Printf("range: %d  level: %d\n", ledRange, level)
	sendCommandToArduino(fmt.Sprintf("flash range %d", ledRange))
	sendCommandToArduino(fmt.Sprintf("flash level %d", level))
}

func closeCurrentPort() {
	myWin.spMutex.Lock()
	if myWin.serialPort != nil {
		err := myWin.serialPort.Close()
		if err != nil {
			log.Println(fmt.Errorf("closeCurrentPort(): %w", err))
		}
		myWin.serialPort = nil
		gpsData = GPSdata{}
		updateStatusLine(gpsData)
		addToTextOutDisplay(fmt.Sprintf("%s has been closed by user", myWin.comPortName))
		log.Printf("%s has been closed by user", myWin.comPortName)
		myWin.comPortInUse.Text = "Serial port open: none"
		myWin.comPortInUse.Refresh()
	} else {
		addToTextOutDisplay("There is no open serial port")
		log.Println("There is no open serial port")
	}
	myWin.spMutex.Unlock()
}

//func setKeepLogFileFlag(checked bool) {
//	myWin.keepLogFile = checked
//}

func sendCommandToArduino(extCmd string) {
	var cmdGiven string
	if extCmd != "" {
		cmdGiven = extCmd
	} else {
		cmdGiven = myWin.cmdEntry.Text
	}

	// Calculate checksum
	checkSum := byte(0)
	for _, char := range cmdGiven {
		checkSum ^= byte(char)
	}

	cmdGiven += fmt.Sprintf("*%02X\r\n", checkSum)
	myWin.spMutex.Lock()
	if myWin.serialPort != nil {
		_, err := myWin.serialPort.Write([]byte(cmdGiven))
		if err != nil {
			errMsg := fmt.Errorf("%w", err)
			log.Println(errMsg.Error())
		}
	}
	myWin.spMutex.Unlock()
}

func showCommandHelp() {
	helpWin := myWin.App.NewWindow("Commands")
	helpWin.Resize(fyne.Size{Height: 700, Width: 700})
	scrollableText := container.NewVScroll(widget.NewRichTextWithText(cmdText))
	helpWin.SetContent(scrollableText)
	helpWin.Show()
	helpWin.CenterOnScreen()
}

func handleComPortSelection(value string) {
	myWin.spMutex.Lock()
	defer myWin.spMutex.Unlock()
	if myWin.serialPort != nil {
		// There is a port already in use. We will close it.
		err := myWin.serialPort.Close()
		if err != nil {
			msg := fmt.Sprintf("Attempt to close %s failed.", myWin.comPortName)
			log.Println(msg)
			addToTextOutDisplay(msg)
			return
		}
		msg := fmt.Sprintf("The currently active serial port (%s) was closed.", myWin.comPortName)

		gpsData = GPSdata{}
		updateStatusLine(gpsData)
		myWin.statusLine.Refresh()

		myWin.selectComPort.Refresh()

		myWin.comPortName = ""
		addToTextOutDisplay(msg)
		log.Println(msg)
		msg = fmt.Sprintf("Make a new serial port selection.")
		addToTextOutDisplay(msg)
		log.Println(msg)
		myWin.comPortInUse.SetText("Serial port open: " + "none")
		return
	}

	myWin.comPortName = value

	if myWin.comPortName != "" {
		serialPort, err := openSerialPort(myWin.comPortName, myWin.curBaudRate)
		myWin.serialPort = serialPort
		if err != nil {
			msg := fmt.Sprintf("Attempt to open %s failed.", myWin.comPortName)
			addToTextOutDisplay(msg)
			log.Println(msg)
			return
		} else {
			msg := fmt.Sprintf("%s was opened successfully.", myWin.comPortName)
			addToTextOutDisplay(msg)
			log.Println(msg)
		}
		myWin.comPortInUse.SetText("Serial port open: " + value)
	}
}

func clearSerialOutputDisplay() {
	myWin.textOut = []string{""}
	myWin.textOutDisplay.Refresh()
}

func getInitialText() []string {
	var newLine string
	var ans []string

	newLine = "... serial output will appear here once the Arduino starts up."
	ans = append(ans, newLine)
	//newLine = fmt.Sprintf("... the serial port parameters: 8,N,1 and %d baudrate.", baudrate)
	//addToTextOutDisplay(newLine)

	return ans
}

func makeStatusLine(app *Config) *fyne.Container {
	app.statusStatus = canvas.NewText("Status: not available", nil)
	app.statusStatus.Alignment = fyne.TextAlignLeading

	app.latitudeStatus = canvas.NewText("Latitude: not available", nil)

	app.longitudeStatus = canvas.NewText("Longitude: not available", nil)

	app.altitudeStatus = canvas.NewText("Altitude: not available", nil)

	app.dateTimeStatus = canvas.NewText("UTC date/time: not available", nil)

	// 5 items with only 4 columns so that dateTime appears on second line
	ans := container.NewGridWithColumns(4,
		app.statusStatus, app.latitudeStatus, app.longitudeStatus, app.altitudeStatus, app.dateTimeStatus)
	return ans
}

func showHelp() {
	//fmt.Println("User asked for help display")
	helpWin := myWin.App.NewWindow("IOTA GFT help")
	helpWin.Resize(fyne.Size{Height: 400, Width: 700})
	scrollableText := container.NewVScroll(widget.NewRichTextWithText(helpText))
	helpWin.SetContent(scrollableText)
	helpWin.Show()
	helpWin.CenterOnScreen()
}

func showMsg(title string, msg string, height, width float32) {
	msgWin := myWin.App.NewWindow(title)
	msgWin.Resize(fyne.Size{Height: height, Width: width})
	scrollableText := container.NewVScroll(widget.NewRichTextWithText(msg))
	msgWin.SetContent(scrollableText)
	msgWin.Show()
	msgWin.CenterOnScreen()
	msgWin.RequestFocus()
}
