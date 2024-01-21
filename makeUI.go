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

func (app *Config) makeUI() {

	app.statusLine = makeStatusLine(app)

	//app.serDataLines = getInitialText(200)
	app.serDataLines = []string{}

	topItem := container.NewVBox(app.statusLine)

	// Compose the left hand column element of the main Border layout
	leftItem := container.NewVBox()
	leftItem.Add(widget.NewButton("Help", func() { showHelp() }))
	leftItem.Add(canvas.NewText("=====================", color.NRGBA{R: 180, A: 255}))

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
	app.comPortInUse = widget.NewLabel("Com port: none")
	leftItem.Add(app.comPortInUse)
	leftItem.Add(widget.NewSeparator())

	leftItem.Add(layout.NewSpacer())
	leftItem.Add(widget.NewButton("Clear output", func() { clearSerialOutputDisplay() }))
	app.autoScroll = widget.NewCheck("Auto-scroll enabled", func(bool) {})
	app.autoScroll.SetChecked(true)
	leftItem.Add(app.autoScroll)

	// Compose bottom element of the main Border layout

	bottomEntry := container.NewBorder( // top, bottom, left, right, center
		nil,
		nil,
		canvas.NewText("Enter cmd:", nil),
		widget.NewButton("Send", func() {}),
		widget.NewEntry(),
	)
	bottomItem := container.NewVBox(widget.NewSeparator(), bottomEntry)

	app.serDataLines = getInitialText()
	app.serOutList = widget.NewList(
		func() int {
			return len(app.serDataLines)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(app.serDataLines[i])
		})

	// We will build the GUI around a Border container.

	content := container.NewBorder(
		topItem,
		bottomItem,
		leftItem,
		nil,
		app.serOutList)

	app.MainWindow.SetContent(content)
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
			addToSerialOutputDisplay(msg)
			return
		}
		msg := fmt.Sprintf("%s was closed.", myWin.comPortName)
		addToSerialOutputDisplay(msg)

	}
	serialPort, err := openSerialPort(myWin.comPortName, myWin.curBaudRate)
	myWin.serialPort = serialPort
	if err != nil {
		msg := fmt.Sprintf("Attempt to open %s failed.", myWin.comPortName)
		addToSerialOutputDisplay(msg)
		return
	} else {
		msg := fmt.Sprintf("%s was opened successfully.", myWin.comPortName)
		addToSerialOutputDisplay(msg)
	}
	fmt.Println("com port", value, "was selected")
	myWin.comPortInUse.SetText("Com port: " + value)
}
func clearSerialOutputDisplay() {
	myWin.serDataLines = []string{""}
	myWin.serOutList.Refresh()
}

func addToSerialOutputDisplay(msg string) {
	myWin.serDataLines = append(myWin.serDataLines, msg)
	myWin.serOutList.Refresh()
}

func getInitialText() []string {
	var newLine string
	var ans []string

	newLine = "... serial output will appear here once the Arduino starts up."
	ans = append(ans, newLine)

	return ans
}
