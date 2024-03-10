package main

import (
	_ "embed"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"go.bug.st/serial"
	"gonum.org/v1/plot/font"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

const (
	MaxSerialDataLines = 100_000
	Version            = "1.0.2"
)

type TickStamp struct {
	utcTime         string // UTC time when P sentence occurred
	gpsTime         string // GPS time when P event occurred
	runningTickTime int64  // runningTickTime at P event
	tickTime        int64  // tickTime reported at P event
}

type FlashEdge struct {
	edgeTime int64 // This is the runningTickTime at P event
	on       bool
}

type OnePPSdata struct {
	startTime       string      // UTC time of first valid 1pps reading
	runningTickTime int64       // sum of all P event tickTime
	tickStamp       []TickStamp // contains info for all P events
	pDelta          []int64     // delta tickTime for all P events
}

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
	gpsUtcOffset  string
	hour          int
	minute        int
	second        int
	year          int
	month         int
	day           int
	gpsTimestamp  string
	utcTimestamp  string
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

var onePPSdata OnePPSdata

var gpsData GPSdata

var flashEdges []FlashEdge

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

//func calc1ppsStats() {
//	thresh := int64(4)
//	deltas := onePPSdata.pDelta[1:]
//	for i := 1; i < len(deltas); i++ {
//		deltaDelta := deltas[i] - deltas[i-1]
//		if !(-thresh < deltaDelta && deltaDelta < thresh) {
//			fmt.Printf("At entry %d, found tick count change of %d\n", i, deltaDelta)
//			fmt.Printf("UTC time: %s\n", onePPSdata.tickStamp[i].utcTime)
//		}
//	}
//}

func calcFlashEdgeTimes() {
	nEdges := len(flashEdges)
	fmt.Printf("%d flash edges available.\n", nEdges)
	for i := range flashEdges {
		for j := 0; j < len(onePPSdata.tickStamp); j++ {
			// Find the onePPS time stamp that precedes the flash edge - we go past it, then back up 1 step
			if onePPSdata.tickStamp[j].runningTickTime > flashEdges[i].edgeTime {
				leftPoint := j - 1
				rightPoint := j
				for k := 0; k < 11; k++ {
					newTimestamp := interpolateTimestamp(
						flashEdges[i].edgeTime,
						onePPSdata.tickStamp[leftPoint-k].runningTickTime,
						onePPSdata.tickStamp[rightPoint+k].runningTickTime,
						onePPSdata.tickStamp[leftPoint-k].utcTime,
						onePPSdata.tickStamp[rightPoint+k].utcTime)
					fmt.Println(newTimestamp)
				}
				break
			}
		}
	}
}

func interpolateTimestamp(flashTime, t1, t2 int64, s1, s2 string) string {
	// Calculate seconds since start
	seconds1 := float64(calcDeltaSeconds(onePPSdata.startTime, s1))
	seconds2 := float64(calcDeltaSeconds(onePPSdata.startTime, s2))

	// Convert tick times to float64
	time1 := float64(t1)
	time2 := float64(t2)

	// Calculate slope of seconds versus ticks
	a := (seconds2 - seconds1) / (time2 - time1)

	// Calculate f(flashTime)  output is time (in seconds) relative to seconds1
	deltaTsecs := a * float64(flashTime-t1)

	interpolatedTimestamp := calcAdderToTimestamp(s1, deltaTsecs)
	return interpolatedTimestamp
}

func testPngDisplay() {
	//calc1ppsStats()

	calcFlashEdgeTimes()

	buildPlot() // Writes judy.png in current working directory

	pngWin := myWin.App.NewWindow("Test png display")
	pngWin.Resize(fyne.Size{Height: 450, Width: 1400})

	testImage := canvas.NewImageFromFile("judy.png")
	pngWin.SetContent(testImage)
	pngWin.CenterOnScreen()
	pngWin.Show()

}

func buildPlot() {

	n := len(onePPSdata.tickStamp)
	myPts := make(plotter.XYs, n)
	for i := range myPts {
		timeInSeconds := calcDeltaSeconds(onePPSdata.startTime, onePPSdata.tickStamp[i].utcTime)
		myPts[i].X = float64(timeInSeconds)
		myPts[i].Y = float64(onePPSdata.tickStamp[i].runningTickTime)
	}

	plot.DefaultFont = font.Font{Typeface: "Liberation", Variant: "Sans", Style: 0, Weight: 3, Size: font.Points(20)}

	plt := plot.New()
	plt.Title.Text = "micro-time versus UTC time"
	plt.X.Label.Text = "elapsed time (seconds)"
	plt.Y.Label.Text = "micro-time (ticks)"

	plotutil.DefaultGlyphShapes[0] = plotutil.Shape(5) // set point shape to filled circle

	err := plotutil.AddScatters(plt, myPts)
	if err != nil {
		panic(err)
	}

	err = plt.Save(20*vg.Inch, 6*vg.Inch, "judy.png")
	if err != nil {
		panic(err)
	}
}
