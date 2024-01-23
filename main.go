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

type GPSdata struct {
	date          string
	timeUTC       string
	latitude      string
	latDirection  string
	longitude     string
	lonDirection  string
	altitude      string
	altitudeUnits string
}

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
	portsAvailable  []string
	autoScroll      *widget.Check
	textOut         []string
	textOutDisplay  *widget.List
	selectComPort   *widget.Select
	comPortName     string
	curBaudRate     int
	serialPort      serial.Port
	lastPvalue      int64
	gpggaCheckBox   *widget.Check
	gprmcCheckBox   *widget.Check
	gpdtmCheckBox   *widget.Check
	pubxCheckBox    *widget.Check
	pCheckBox       *widget.Check
	modeCheckBox    *widget.Check
	cmdEntry        *widget.Entry
}

//go:embed help.txt
var helpText string

var gpsData GPSdata

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
	scanForComPorts()

	// Start the application go routine where all the work is done

	go runApp(&myWin)

	// show and run the GUI
	myWin.MainWindow.ShowAndRun()

	// We're closing, so clean up any allocated resources
	if myWin.serialPort != nil {
		err := myWin.serialPort.Close()
		if err != nil {
			log.Fatal("While closing serial port got:", err)
		}
	}
}

func scanForComPorts() {
	ports, err := getSerialPortsList()
	if err != nil {
		addToTextOutDisplay("Fatal err: could not get list of available com ports")
	}

	var realPorts []string
	for _, port := range ports {
		sp, err := openSerialPort(port, 250000)
		if err == nil {
			// It's an actual attached and active port
			_ = sp.Close()

			// But check for duplicate names - duplicate names are generated
			// whenever a com port is disconnected and reconnected (for some unknown reason)
			duplicate := false
			for _, p := range realPorts {
				if p == port {
					duplicate = true
				}
			}

			if !duplicate {
				realPorts = append(realPorts, port)
			}
		}
	}

	//realPorts = append(realPorts, "COM?") // For testing purposes only
	//fmt.Println("Current ports list:", ports)

	myWin.portsAvailable = realPorts
	myWin.selectComPort.SetOptions(myWin.portsAvailable)

	if len(myWin.portsAvailable) == 0 {
		myWin.selectComPort.ClearSelected()
	}
	myWin.selectComPort.Refresh()

	if len(myWin.portsAvailable) == 1 {
		myWin.comPortName = myWin.portsAvailable[0]
		myWin.comPortInUse.SetText("Port in use: " + myWin.portsAvailable[0])
		myWin.selectComPort.SetSelectedIndex(0) // Note: this acts as though the user clicked on this entry
	}
}

func addToTextOutDisplay(msg string) {
	myWin.textOut = append(myWin.textOut, msg)
	myWin.textOutDisplay.Refresh()
	if myWin.autoScroll.Checked {
		myWin.textOutDisplay.ScrollToBottom()
	}
}

func initializeStartingWindow(myWin *Config) {
	myWin.App = app.New()
	myWin.MainWindow = myWin.App.NewWindow("IOTA GFT app")
	myWin.MainWindow.Resize(fyne.Size{Height: 600, Width: 1000})
	myWin.MainWindow.SetMaster() // As 'master', if the window is closed, the application quits.
	myWin.MainWindow.CenterOnScreen()
}
