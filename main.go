package main

import (
	_ "embed"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"go.bug.st/serial"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	MaxSerialDataLines = 100_000
	Version            = "1.0.0"
)

type GPSdata struct {
	date          string
	timeUTC       string
	latitude      string
	latDirection  string
	longitude     string
	lonDirection  string
	altitude      string
	altitudeUnits string
	status        string
}

type Config struct {
	App             fyne.App
	MainWindow      fyne.Window
	HelpViewer      *widget.RichText
	statusLine      *fyne.Container
	statusStatus    *canvas.Text
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
	spMutex         sync.Mutex
	lastPvalue      int64
	logCheckBox     *widget.Check
	gpggaCheckBox   *widget.Check
	gprmcCheckBox   *widget.Check
	gpdtmCheckBox   *widget.Check
	pubxCheckBox    *widget.Check
	pCheckBox       *widget.Check
	modeCheckBox    *widget.Check
	cmdEntry        *widget.Entry
	logFile         *os.File
	keepLogFile     bool
}

//go:embed help.txt
var helpText string

//go:embed cmd.txt
var cmdText string

var gpsData GPSdata

var myWin Config

var logfilePath string

// The following default baudrate can be changed by a command line argument
var baudrate = 250000

func main() {

	// A non-standard baudrate (which is normally 250000) can be specified on the command line
	//fmt.Println(len(os.Args), os.Args)
	if len(os.Args) > 1 {
		cmdLineBaudrate, err := strconv.Atoi(os.Args[1])
		if (err != nil) || (baudrate < 0) {
			fmt.Println("Baudrate given on command line was not a positive integer")
			os.Exit(911)
		} else {
			if baudrate != 250000 {
				fmt.Printf("Cmdline changed baudrate from standard 250000 to: %d", cmdLineBaudrate)
				baudrate = cmdLineBaudrate
			}
		}
	}

	// Form a unique name for the log file from the working directory (where the app has been run from)
	// and current time. There is a button that the user can to change the name of the log file. That
	// action is deferred until the app closes.
	t := time.Now().UTC()
	timestamp := t.Format(time.RFC822Z)
	// Replace spaces with - (to make a more friendly file name)
	timestamp = strings.Replace(timestamp, " ", "_", -1)
	timestamp = strings.Replace(timestamp, ":", "_", -1)
	timestamp = timestamp[0:len(timestamp)-6] + "UTC"
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Println("os.Getwd() failed to return working directory")
		os.Exit(911)
	}

	// Form the full path to the logfile
	logfilePath = fmt.Sprintf("%s\\LOG_GFT_%s.txt", workDir, timestamp)

	// create and open the logFile
	logFile, err1 := os.Create(logfilePath)
	if err1 != nil {
		log.Fatal(err1)
	}
	myWin.logFile = logFile

	// close the file for sure when app exits
	defer func(logFile *os.File) {
		err := logFile.Close()
		if err != nil {
		}
	}(logFile)

	initializeStartingWindow(&myWin)

	// Build the GUI
	myWin.makeUI()

	defer deleteLogfile()

	newLine := fmt.Sprintf("... the serial port will be opened at 8,N,1 and %d baudrate.", baudrate)
	addToTextOutDisplay(newLine)

	if baudrate != 250000 {
		newLine = fmt.Sprintf("... a non-standard baudrate of %d has been specified in the command line.", baudrate)
		addToTextOutDisplay(newLine)
	}

	newLine = fmt.Sprintf("Log file @ %s", logfilePath)
	addToTextOutDisplay(newLine)

	// Find available com ports, fill in the drop-down list of available serial
	// ports and, if there is exactly one comport, open it at the default baudrate.
	scanForComPorts()

	// Start the application go routine where all the work is done

	go runApp(&myWin)

	// show and run the GUI
	myWin.MainWindow.ShowAndRun()

	// We're closing, so clean up any allocated resources
	myWin.spMutex.Lock()
	if myWin.serialPort != nil {
		err := myWin.serialPort.Close()
		if err != nil {
			log.Fatal("While closing serial port got:", err)
		}
	}
	myWin.spMutex.Unlock()
}

func addToTextOutDisplay(msg string) {
	// Write every msg added to the text out panel to the log file
	//_, fileErr := myWin.logFile.WriteString(msg + "\n")
	//if fileErr != nil {
	//	fmt.Println(fmt.Errorf("addToTextOutDisplay(): %w", fileErr))
	//}

	if len(myWin.textOut) >= MaxSerialDataLines {
		myWin.textOut = []string{""}
	}
	myWin.textOut = append(myWin.textOut, msg)
	myWin.textOutDisplay.Refresh()
	if myWin.autoScroll.Checked {
		myWin.textOutDisplay.ScrollToBottom()
	}
}

func initializeStartingWindow(myWin *Config) {
	myWin.App = app.New()
	myWin.MainWindow = myWin.App.NewWindow("IOTA GFT " + Version)
	myWin.MainWindow.Resize(fyne.Size{Height: 600, Width: 1100})
	myWin.MainWindow.SetMaster() // As 'master', if the window is closed, the application quits.
	myWin.MainWindow.CenterOnScreen()
}

func deleteLogfile() {
	//fmt.Println("State of logCheckBox: ", checked)
	if !myWin.keepLogFile {
		//fmt.Println("Deleting log file")
		//myWin.logCheckBox.Disable()
		filePath := myWin.logFile.Name()
		err := myWin.logFile.Close()
		if err != nil {
			fmt.Println(fmt.Errorf("deleteLogfile(): %w", err))
		}
		myWin.logFile = nil
		err = os.Remove(filePath)
		if err != nil {
			fmt.Println(fmt.Errorf("deleteLogfile(): %w", err))
		}
		//fmt.Println("Log file deleted")
	}
}
