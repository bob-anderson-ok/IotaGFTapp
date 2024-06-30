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
	"math"
	"net"
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
	Version            = "1.1.3"
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
	unixTime      int64
	nextUnixTime  int64 // We use this to detect missing 1pps pulses
}

type Config struct {
	App                  fyne.App
	SharpCapConn         net.Conn
	SharpCapAvailable    bool
	flashIntensitySlider *widget.Slider
	MainWindow           fyne.Window
	HelpViewer           *widget.RichText
	statusLine           *fyne.Container
	statusStatus         *canvas.Text
	latitudeStatus       *canvas.Text
	longitudeStatus      *canvas.Text
	altitudeStatus       *canvas.Text
	dateTimeStatus       *canvas.Text
	comPortInUse         *widget.Label
	portsAvailable       []string
	autoScroll           *widget.Check
	textOut              []string
	textOutDisplay       *widget.List
	selectComPort        *widget.Select
	comPortName          string
	curBaudRate          int
	serialPort           serial.Port
	spMutex              sync.Mutex
	lastPvalue           int64
	logCheckBox          *widget.Check
	gpggaCheckBox        *widget.Check
	gprmcCheckBox        *widget.Check
	gpdtmCheckBox        *widget.Check
	pubxCheckBox         *widget.Check
	pCheckBox            *widget.Check
	modeCheckBox         *widget.Check
	cmdEntry             *widget.Entry
	pathEntry            *widget.Entry
	utcEventTime         *widget.Entry
	eventDateTime        time.Time
	leaderStartTime      int64
	firstFlashTime       int64
	secondFlashTime      int64
	endOfRecording       int64
	recordingLength      *widget.Entry
	recordingDuration    float64
	logFilePath          string
	logFile              *os.File
	flashEdgeLogfilePath string
	flashEdgeLogfile     *os.File
	keepLogFile          bool
	gotFirst1PPS         bool
	utcStartArmed        bool
	pastLeader           bool
	pastFlashOne         bool
	pastFlashTwo         bool
	pastEnd              bool
	armUTCbutton         *widget.Button
}

//go:embed help.txt
var helpText string

//go:embed recordingLengthError.txt
var recordingLengthError string

//go:embed utcTimeError.txt
var utcTimeError string

//go:embed cmd.txt
var cmdText string

//go:embed sharpCapError.txt
var sharpCapErr string

var onePPSdata OnePPSdata

var gpsData GPSdata

var flashEdges []FlashEdge

var myWin Config

var logfilePath string
var flashEdgeLogfilePath string

// The following default baudrate can be changed by a command line argument
var baudrate = 250000

const MSGLEN = 1000

const (
	ServerHost   = "127.0.0.1"
	IotaGFTPort  = "33001"
	SharpCapPort = "33000"
	ServerType   = "tcp"
)

func makeMsg(msg string) []byte {
	// Pad msg with spaces to make a fixed length message of size MSGLEN
	paddedMsg := make([]byte, MSGLEN)
	for i := 0; i < len(msg); i++ {
		paddedMsg[i] = msg[i]
	}
	for i := len(msg); i < MSGLEN; i++ {
		paddedMsg[i] = ' '
	}
	return paddedMsg
}

func msgTrim(msg string) string {
	return strings.TrimSpace(msg)
}

func getResponse(conn net.Conn, cmd string) string {
	_, err := conn.Write(makeMsg(cmd))
	if err != nil {
		fmt.Println("Error writing:", err.Error())
	}
	buffer := make([]byte, MSGLEN)
	_, err = conn.Read(buffer) // Get response - blocks until MSGLEN bytes have been received
	if err != nil {
		fmt.Println("Error reading:", err.Error())
	}
	return msgTrim(string(buffer[:]))
}

func server() {
	// establish connection
	server, err := net.Listen(ServerType, ServerHost+":"+IotaGFTPort)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	defer server.Close()
	fmt.Println("Listening on " + ServerHost + ":" + IotaGFTPort)
	fmt.Println("Waiting for client...")
	for {
		connection, err := server.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		//fmt.Println("client connected")
		go processClient(connection)
	}
}

func processClient(connection net.Conn) {
	buffer := make([]byte, 1024)
	bytesRead := 0
	chunks := make([]byte, 0)
	for {
		mLen, err := connection.Read(buffer)
		if err != nil {
			fmt.Println("Error reading:", err.Error())
			os.Exit(1)
		}
		bytesRead += mLen
		chunks = append(chunks, buffer[:mLen]...)
		if bytesRead == MSGLEN {
			break
		}
	}
	fmt.Println("Received: ", strings.TrimSpace(string(buffer[:bytesRead])))
	connection.Close()
}

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

	// Form a unique name for the log file from the working directory.
	workDir := getWorkDir()

	initializeStartingWindow(&myWin)

	// Build the GUI
	myWin.makeUI()

	myWin.utcStartArmed = false
	myWin.pastLeader = false
	myWin.pastFlashOne = false
	myWin.pastFlashTwo = false
	myWin.pastEnd = false

	createLogAndFlashEdgeFiles(workDir)

	//defer deleteLogfile()

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

	go server()

	checkSharpCapAvailability()

	// show and run the GUI
	myWin.MainWindow.ShowAndRun()

	// We're closing, so clean up any allocated resources
	myWin.spMutex.Lock()
	if myWin.serialPort != nil {
		err := myWin.serialPort.Close()
		if err != nil {
			fmt.Println("While closing serial port got:", err)
		}
	}
	myWin.spMutex.Unlock()

	if myWin.SharpCapAvailable {
		myWin.SharpCapConn.Close()
	}

	//calcFlashEdgeTimes() // These get written to the flashEdgeLogfile

	_ = myWin.logFile.Close()
	_ = myWin.flashEdgeLogfile.Close()
}

func getWorkDir() string {
	//t := time.Now().UTC()
	//timestamp := t.Format(time.RFC822Z)
	//// Replace spaces with - (to make a more friendly file name)
	//timestamp = strings.Replace(timestamp, " ", "_", -1)
	//timestamp = strings.Replace(timestamp, ":", "_", -1)
	//timestamp = timestamp[0:len(timestamp)-6] + "UTC"
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Println("os.Getwd() failed to return working directory")
		os.Exit(911)
	}
	return workDir
}

func checkSharpCapAvailability() {
	var err error
	myWin.SharpCapConn, err = net.Dial(ServerType, ServerHost+":"+SharpCapPort)
	if err != nil {
		showMsg("SharpCap unavailable", sharpCapErr, 600, 550)
		myWin.SharpCapAvailable = false
		fmt.Println("SharpCap not running") // TODO Popup a message window
	} else {
		myWin.SharpCapAvailable = true
	}
}

// exists returns whether the given file or directory exists
//func exists(path string) (bool, error) {
//	_, err := os.Stat(path)
//	if err == nil {
//		return true, nil
//	}
//	if os.IsNotExist(err) {
//		return false, nil
//	}
//	return false, err
//}

func createLogAndFlashEdgeFiles(workDir string) bool {
	// Form the full path to the standard logfile
	//logfilePath = fmt.Sprintf("%s/LOG_GFT_%s.txt", workDir, timestamp)
	logfilePath = fmt.Sprintf("%s/LOG_GFT.txt", workDir)
	//flashEdgeLogfilePath = fmt.Sprintf("%s/FLASH_EDGE_TIMES_%s.txt", workDir, timestamp)
	flashEdgeLogfilePath = fmt.Sprintf("%s/FLASH_EDGE_TIMES.txt", workDir)
	myWin.logFilePath = logfilePath
	myWin.flashEdgeLogfilePath = flashEdgeLogfilePath

	// create and open the logFile
	logFile, err1 := os.Create(logfilePath)
	if err1 != nil {
		fmt.Println("in createLogAndFlashEdgeFiles:", err1)
		return false
	}
	myWin.logFile = logFile

	// create and open the flash edge logfile
	flashLogFile, err1 := os.Create(flashEdgeLogfilePath)
	if err1 != nil {
		fmt.Println("in createLogAndFlashEdgeFiles:", err1)
		return false
	}
	myWin.flashEdgeLogfile = flashLogFile
	return true
}

func addToTextOutDisplay(msg string) {

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

//func deleteLogfile() {
//	filePath := myWin.logFile.Name()
//	fmt.Println("Deleting log file:", filePath)
//	err := myWin.logFile.Close()
//	if err != nil {
//		fmt.Println(fmt.Errorf("deleteLogfile(): %w", err))
//	}
//	myWin.logFile = nil
//	err = os.Remove(filePath)
//	if err != nil {
//		fmt.Println(fmt.Errorf("deleteLogfile(): %w", err))
//	}
//}

//func deleteEdgeTimesFile() {
//	filePath := myWin.flashEdgeLogfile.Name()
//	fmt.Println("Deleting edge times file:", filePath)
//	err := myWin.flashEdgeLogfile.Close()
//	if err != nil {
//		fmt.Println(fmt.Errorf("deleteEdgeTimesFile(): %w", err))
//	}
//	myWin.flashEdgeLogfile = nil
//	err = os.Remove(filePath)
//	if err != nil {
//		fmt.Println(fmt.Errorf("deleteEdgeTimesFile(): %w", err))
//	}
//}

func calcFlashEdgeTimes() {
	//nEdges := len(flashEdges)
	//fmt.Printf("%d flash edges available.\n", nEdges)
	for i := range flashEdges {
		for j := 0; j < len(onePPSdata.tickStamp); j++ {
			// Find the onePPS time stamp that precedes the flash edge - we go past it, then back up 1 step
			if onePPSdata.tickStamp[j].runningTickTime > flashEdges[i].edgeTime {
				leftPoint := j - 1
				rightPoint := j
				newTimestamp := interpolateTimestamp(
					flashEdges[i].edgeTime,
					onePPSdata.tickStamp[leftPoint].runningTickTime,
					onePPSdata.tickStamp[rightPoint].runningTickTime,
					onePPSdata.tickStamp[leftPoint].utcTime,
					onePPSdata.tickStamp[rightPoint].utcTime)

				edgeStr := ""
				if flashEdges[i].on {
					edgeStr = fmt.Sprintf("%d on  %s\n", i+1, newTimestamp+"Z") // Count flash edges starting from 1
				} else {
					edgeStr = fmt.Sprintf("%d off %s\n", i+1, newTimestamp+"Z")
				}
				_, fileErr := myWin.flashEdgeLogfile.WriteString(edgeStr)
				// TODO Remove the following print
				fmt.Println(edgeStr)
				if fileErr != nil {
					fmt.Println(fmt.Errorf("calcFlashEdgeTimes(): %w", fileErr))
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

func show1ppsHistory() {

	buildPlot() // Writes ppsHistory.png in current working directory

	pngWin := myWin.App.NewWindow("1pps history")
	pngWin.Resize(fyne.Size{Height: 450, Width: 1400})

	testImage := canvas.NewImageFromFile("ppsHistory.png")
	pngWin.SetContent(testImage)
	pngWin.CenterOnScreen()
	pngWin.Show()
}

func validRecordingTime() bool {
	var textGiven = myWin.recordingLength.Text
	value, err := strconv.ParseFloat(textGiven, 64)
	if err != nil {
		return false
	}
	if value <= 0.0 {
		return false
	}
	myWin.recordingDuration = value
	//fmt.Println("recording length (sec):", textGiven)
	return true
}

func validUTCtime() bool {
	var textGiven = myWin.utcEventTime.Text
	utcTime, err := time.Parse(time.DateTime, textGiven)
	if err != nil {
		return false
	}
	fmt.Println("utc date/time entered:", utcTime)
	myWin.eventDateTime = utcTime
	return true
}

func calculateStartTime(delta int64) string {
	exposureStr := getResponse(myWin.SharpCapConn, "exposure")
	fmt.Println("Rcvd:", exposureStr)
	if exposureStr == "No camera selected" {
		showMsg("SharpCap error", "\nNo camera selected!\n", 200, 200)
		return "No camera selected"
	}
	exposureMs, err := strconv.ParseFloat(exposureStr, 64)
	if err != nil {
		showMsg("Format error", err.Error(), 200, 200)
		return "Exposure string invalid"
	}
	//fmt.Println(exposureMs)
	readingsPerSecond := 1000 / exposureMs
	fmt.Println(readingsPerSecond, "readings per second")
	neededFlashTime := int(math.Ceil(10 / readingsPerSecond))
	//if neededFlashTime < 1.0 {
	//	neededFlashTime = 1
	//} else {
	//	neededFlashTime = math.Ceil(neededFlashTime)
	//}
	// Set the flash duration (seconds)
	cmd := fmt.Sprintf("flash duration %d", neededFlashTime)
	sendCommandToArduino(cmd)

	unixTimeNow := gpsData.unixTime

	flashTime := int64(neededFlashTime) // seconds
	startTime := unixTimeNow + delta
	d := unixTimeNow - startTime
	fmt.Println("time now:", unixTimeNow)
	fmt.Println("time until start of acquisition:", -d)
	if d < 0 {
		myWin.leaderStartTime = startTime
		myWin.firstFlashTime = myWin.leaderStartTime + flashTime
		myWin.secondFlashTime = myWin.firstFlashTime + flashTime + int64(myWin.recordingDuration)
		myWin.endOfRecording = myWin.secondFlashTime + 3*flashTime
		return "ok"
	} else {
		return "start time is in the past"
	}
}

func armUTCstart() {
	//fmt.Println("Arm UTC start clicked")
	if !myWin.utcStartArmed {
		if !myWin.SharpCapAvailable {
			checkSharpCapAvailability()
			if !myWin.SharpCapAvailable {
				return
			}
		}

		if !validRecordingTime() {
			showMsg("Invalid recording time", recordingLengthError, 250, 400)
			return
		}

		utcText := myWin.utcEventTime.Text
		fmt.Println("UTC event time:", utcText)

		var result string
		myWin.pastLeader = false
		myWin.pastFlashOne = false
		myWin.pastFlashTwo = false
		myWin.pastEnd = false

		workDir := getWorkDir()
		createLogAndFlashEdgeFiles(workDir)

		if utcText == "" {
			fmt.Println("Start 10 seconds from now")
			result = calculateStartTime(10)
		} else {
			if !validUTCtime() {
				showMsg("Invalid UTC date/time", utcTimeError, 250, 400)
				return
			}
			result = calculateStartTime(0)
		}

		if result != "ok" {
			showMsg("Start time error", "\n"+result+"\n", 250, 400)
			return
		}

		processFlashIntensitySliderChange(myWin.flashIntensitySlider.Value)

		myWin.armUTCbutton.SetText("UTC start armed and active")
		myWin.armUTCbutton.Importance = widget.SuccessImportance
		myWin.utcStartArmed = true
	} else {
		myWin.utcStartArmed = false
		myWin.armUTCbutton.Importance = widget.MediumImportance
		myWin.armUTCbutton.SetText("Arm UTC start")
	}
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

	err = plt.Save(20*vg.Inch, 6*vg.Inch, "ppsHistory.png")
	if err != nil {
		panic(err)
	}
}
