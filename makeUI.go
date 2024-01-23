package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"strconv"
)

type InputField struct {
	widget.Entry
	inputText      widget.Entry
	customFunction func()
}

func NewInputField() *InputField {
	i := &InputField{
		Entry: widget.Entry{},
		inputText: widget.Entry{
			DisableableWidget: widget.DisableableWidget{},
			Text:              "",
			TextStyle:         fyne.TextStyle{},
			PlaceHolder:       "",
			OnChanged:         nil,
			OnSubmitted:       nil,
			Password:          false,
			MultiLine:         false,
			Wrapping:          0,
			Scroll:            0,
			Validator:         nil,
			CursorRow:         0,
			CursorColumn:      0,
			OnCursorChanged:   nil,
			ActionItem:        nil,
		},
		customFunction: func() { fmt.Println("customFunction ran") },
	}
	i.ExtendBaseWidget(i)
	return i
}

//func (m *myEntry) TypedKey(key fyne.KeyEvent) {
//	if key.Name == "Return" {
//		fmt.Println("Return typed in cmd entry")
//	} else {
//		m.Entry.TypedKey(&key)
//	}
//}

func (app *Config) makeUI() {

	app.statusLine = makeStatusLine(app)

	app.textOut = []string{}

	topItem := container.NewVBox(app.statusLine)

	// Compose the left hand column element of the main Border layout
	leftItem := container.NewVBox()
	leftItem.Add(widget.NewButton("Help", func() { showHelp() }))
	leftItem.Add(canvas.NewText("=======================", color.NRGBA{R: 180, A: 255}))

	entryField := widget.NewEntry()
	app.curBaudRate = 250000
	entryField.SetText(strconv.Itoa(app.curBaudRate))
	baudrateEntry := container.NewBorder( // top, bottom, left, right, center
		nil,
		nil,
		canvas.NewText("baudrate:", nil),
		nil,
		entryField,
	)

	leftItem.Add(baudrateEntry)
	leftItem.Add(widget.NewSeparator())

	leftItem.Add(canvas.NewText("Com ports available", nil))
	var comPorts []string
	app.selectComPort = widget.NewSelect(comPorts, func(value string) { handleComPortSelection(value) })
	leftItem.Add(app.selectComPort)
	app.comPortInUse = widget.NewLabel("Port in use: none")
	leftItem.Add(app.comPortInUse)
	leftItem.Add(widget.NewSeparator())

	leftItem.Add(layout.NewSpacer())
	leftItem.Add(widget.NewButton("Clear output", func() { clearSerialOutputDisplay() }))
	app.autoScroll = widget.NewCheck("Auto-scroll enabled", func(bool) {})
	app.autoScroll.SetChecked(true)
	leftItem.Add(app.autoScroll)

	rightItem := container.NewVBox()
	rightItem.Add(layout.NewSpacer())

	rightItem.Add(widget.NewLabel("Display items"))

	app.gpggaCheckBox = widget.NewCheck("$GPGGA", func(bool) {})
	app.gpggaCheckBox.SetChecked(false)
	rightItem.Add(app.gpggaCheckBox)

	app.gprmcCheckBox = widget.NewCheck("$GPRMC", func(bool) {})
	app.gprmcCheckBox.SetChecked(false)
	rightItem.Add(app.gprmcCheckBox)

	app.gpdtmCheckBox = widget.NewCheck("$GPDTM", func(bool) {})
	app.gpdtmCheckBox.SetChecked(false)
	rightItem.Add(app.gpdtmCheckBox)

	app.pubxCheckBox = widget.NewCheck("$PUBX", func(bool) {})
	app.pubxCheckBox.SetChecked(false)
	rightItem.Add(app.pubxCheckBox)

	app.pCheckBox = widget.NewCheck("P", func(bool) {})
	app.pCheckBox.SetChecked(false)
	rightItem.Add(app.pCheckBox)

	app.modeCheckBox = widget.NewCheck("MODE", func(bool) {})
	app.modeCheckBox.SetChecked(true)
	rightItem.Add(app.modeCheckBox)

	rightItem.Add(layout.NewSpacer())

	// Compose bottom element of the main Border layout

	app.cmdEntry = widget.NewEntry()
	app.cmdEntry.OnSubmitted = func(str string) { sendCommandToArduino() }

	bottomEntry := container.NewBorder( // top, bottom, left, right, center
		nil,
		nil,
		canvas.NewText("Enter cmd:", nil),
		widget.NewButton("Send cmd", func() { sendCommandToArduino() }),
		app.cmdEntry,
	)
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

func sendCommandToArduino() {
	cmdGiven := myWin.cmdEntry.Text
	cmdGiven += "\r\n"
	_, err := myWin.serialPort.Write([]byte(cmdGiven))
	if err != nil {
		errMsg := fmt.Errorf("%w", err)
		fmt.Println(errMsg.Error())
	}
}
func showHelp() {
	fmt.Println("User asked for help display")
	helpWin := myWin.App.NewWindow("Help")
	helpWin.Resize(fyne.Size{Height: 400, Width: 700})
	scrollableText := container.NewVScroll(widget.NewRichTextWithText(helpText))
	helpWin.SetContent(scrollableText)
	helpWin.Show()
	helpWin.CenterOnScreen()
}

func handleComPortSelection(value string) {
	if myWin.serialPort != nil {
		err := myWin.serialPort.Close()
		if err != nil {
			msg := fmt.Sprintf("Attempt to close %s failed.", myWin.comPortName)
			addToTextOutDisplay(msg)
			return
		}
		msg := fmt.Sprintf("The currently active serial port (%s) was closed.", myWin.comPortName)
		addToTextOutDisplay(msg)
		value = ""
	}

	myWin.comPortName = value
	if value == "" {
		myWin.comPortInUse.SetText("Port in use: " + "none")
		return
	}

	serialPort, err := openSerialPort(myWin.comPortName, myWin.curBaudRate)
	myWin.serialPort = serialPort
	if err != nil {
		msg := fmt.Sprintf("Attempt to open %s failed.", myWin.comPortName)
		addToTextOutDisplay(msg)
		return
	} else {
		msg := fmt.Sprintf("%s was opened successfully.", myWin.comPortName)
		addToTextOutDisplay(msg)
	}
	myWin.comPortInUse.SetText("Port in use: " + value)
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

	return ans
}
