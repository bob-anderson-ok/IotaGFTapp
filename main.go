package main

import (
	_ "embed"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"go.bug.st/serial"
	"log"
	"os"
)

const MaxSerialDataLines = 100_000

type Config struct {
	App             fyne.App
	InfoLog         *log.Logger
	ErrorLog        *log.Logger
	MainWindow      fyne.Window
	HelpViewer      *widget.RichText
	statusLine      *fyne.Container
	latitudeStatus  *canvas.Text
	longitudeStatus *canvas.Text
	altitudeStatus  *canvas.Text
	dateTimeStatus  *canvas.Text
	comPortInUse    *widget.Label
	autoScroll      *widget.Check
	serDataLines    []string
	serOutList      *widget.List
	selectComPort   *widget.Select
	comPortName     string
	curBaudRate     int
	serialPort      serial.Port
}

//go:embed help.txt
var helpText string

var myWin Config

func main() {

	initializeStartingWindow(&myWin)

	// create our loggers
	myWin.InfoLog = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	myWin.ErrorLog = log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	// Build the GUI
	myWin.makeUI()

	// Fill in the available com ports and, if there is exactly one comport, open it
	// at the default baudrate
	ports, err := getSerialPortsList()
	if err != nil {
		addToSerialOutputDisplay("Fatal err: could not get list of available com ports")
	}

	//ports = append(ports, "Another comport")  // For testing purposes only

	myWin.selectComPort.SetOptions(ports)

	if len(ports) == 1 {
		myWin.comPortName = ports[0]
		myWin.comPortInUse.SetText("Com port: " + ports[0])
		myWin.selectComPort.SetSelectedIndex(0) // Note: this acts as though the user clicked on this entry
	}

	// Start the application go routine where all the work is done
	go runApp(&myWin)

	// show and run the GUI
	myWin.MainWindow.ShowAndRun()
}

func initializeStartingWindow(myWin *Config) {
	myWin.App = app.New()
	myWin.MainWindow = myWin.App.NewWindow("IOTA GFT app")
	myWin.MainWindow.Resize(fyne.Size{Height: 600, Width: 900})
	myWin.MainWindow.SetMaster() // As 'master', if the window is closed, the application quits.
	myWin.MainWindow.CenterOnScreen()
}
