package main

import (
	"fmt"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func MoveFile(sourcePath, destPath string) error {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("couldn't open source file: %v", err)
	}
	defer inputFile.Close()

	outputFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("couldn't open dest file: %v", err)
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, inputFile)
	if err != nil {
		return fmt.Errorf("couldn't copy to dest from source: %v", err)
	}

	inputFile.Close() // for Windows, close before trying to remove: https://stackoverflow.com/a/64943554/246801

	err = os.Remove(sourcePath)
	if err != nil {
		return fmt.Errorf("couldn't remove source file: %v", err)
	}
	return nil
}

func runApp(myWin *Config) {

	//myWin.App.Preferences().SetString("gpsUtcOffset", "17") // TODO Use this to test change to GpsUtcOffset

	sentenceChan := make(chan string, 1)

	// This runs infinitely, sending each sentence received to sentenceChan. It has a 2-second timeout
	// for dealing with a non-responsive serial port and returns "timeout" as a sentence in that case.
	go getNextSentence(sentenceChan)

	waitingForNestFinish := false

	//var tickCounter int64
	var tickMsg string

	var ans []string
	var checksumString string
	var err error
	var parts []string
	var partsSaved []string
	var nester, nestee string
	const showTickMsg = true
	time.Sleep(2000 * time.Millisecond)
	showMsg("Special test protocol", noUTCtest, 450, 800)
	for {
		if myWin.serialPort != nil {
			// A 'sentence' is everything up to, but not including, a crlf sequence.
			// The last three characters of the 'sentence' are a checksum *xx (even for a 'nest')
			// The checksum has not yet been validated at this point.
			sentence := <-sentenceChan // Block until a sentence is returned by go getNextSentence(sentenceChan)

			if sentence == "timeout" {
				msg := fmt.Sprintf("Serial port %s is not responding.", myWin.comPortName)
				addToTextOutDisplay(msg)
				log.Println(msg)
				continue
			}

			// Always write every sentence to the log file
			if myWin.logFile != nil {
				_, fileErr := myWin.logFile.WriteString(sentence + "\n")
				if fileErr != nil {
					log.Println(fmt.Errorf("runApp(): %w", fileErr))
				}
			}

			if waitingForNestFinish {
				waitingForNestFinish = false
				nestee = "{" + partsSaved[1] + sentence
				ans, checksumString, err = sendSentenceToBeParsed(nestee, ans, err)
				displayEnabledItems(ans, checksumString)
				ans, checksumString, err = sendSentenceToBeParsed(nester, ans, err)
				displayEnabledItems(ans, checksumString)
				continue
			}

			// Test for nested P, E, +, or - flash phrases pulse
			// or NMEA and E (there will be exactly 2 { characters in the 'nest')
			parts = strings.Split(sentence, "{")
			if len(parts) > 2 {
				// We have a 'nested pulse' situation
				//fmt.Println("Nest found:", sentence)
				nester = "{" + parts[2]
				partsSaved = make([]string, len(parts))
				copy(partsSaved, parts)
				waitingForNestFinish = true
				continue
			}

			// This call checks the checksum
			ans, checksumString, err = sendSentenceToBeParsed(sentence, ans, err)
			needTickMsg := false
			if ans[0] == "P" {
				tickMsg = fmt.Sprintf("unixTime %d ", gpsData.unixTime)
				lostPulseCount := gpsData.unixTime - gpsData.nextUnixTime
				// When the PUBX04 gpsUtcOffset changes from 16D to 18, it appears that -2 1pps
				// pulses were lost. We deal with this by only reporting positive lostPulseCount values
				if lostPulseCount > 0 {
					showMsg("PPS error !",
						fmt.Sprintf("\n%d 1pps pulses were lost !!!\n", lostPulseCount), 200, 800)
					log.Printf("%d 1pps pulses were lost\n", lostPulseCount)
					gpsData.nextUnixTime = gpsData.unixTime // catch up so that we can continue testing
				}
				gpsData.nextUnixTime += 1
				// This is where we check for time to do a start recording
				if myWin.utcStartArmed {
					tNow := gpsData.unixTime

					// The test below (>=) could be just == , but we want to be as robust
					// as possible in case a 1pps pulse goes missing that happens to coincide
					// with a scheduled event
					if tNow >= myWin.leaderStartTime && !myWin.pastLeader {
						tickMsg += fmt.Sprint("Starting leader ")
						log.Println(fmt.Sprint("Starting leader "))
						needTickMsg = true
						myWin.pastLeader = true
						if connectToSharpCap() {
							//Example of asking SharpCap to set exposure time
							//fmt.Println(getResponse(myWin.SharpCapConn, "set_exp_seconds 0.5"))
							getResponse(myWin.SharpCapConn, "start")
						} else {
							clearSchedule(myWin)
							goto endSchedule
						}
					}

					if tNow >= myWin.firstFlashTime && !myWin.pastFlashOne {
						tickMsg += fmt.Sprint("Flash one requested")
						log.Println(fmt.Sprint("Flash one requested"))
						needTickMsg = true
						myWin.pastFlashOne = true
						sendCommandToArduino("flash now")
					}

					if tNow >= myWin.secondFlashTime && !myWin.pastFlashTwo {
						tickMsg += fmt.Sprint("Flash two requested")
						log.Println(fmt.Sprint("Flash two requested"))
						myWin.pastFlashTwo = true
						needTickMsg = true
						sendCommandToArduino("flash now")
					}

					if tNow >= myWin.endOfRecording && !myWin.pastEnd {
						myWin.App.Preferences().SetBool("ArmUTCstartTime", false)

						tickMsg += fmt.Sprint("Recording ended\n")
						log.Println(fmt.Sprint("Recording ended"))
						myWin.pastEnd = true
						needTickMsg = true

						var sharpCapPath string
						if connectToSharpCap() {
							getResponse(myWin.SharpCapConn, "stop")
							sharpCapPath = getResponse(myWin.SharpCapConn, "lastfilepath")
						} else {
							clearSchedule(myWin)
							goto endSchedule
						}
						dirPath, _ := filepath.Split(sharpCapPath)
						if !myWin.shutdownCheckBox.Checked {
							showMsg("Path to SharpCap capture folder:", dirPath, 200, 800)
							log.Println(fmt.Sprint("Path to SharpCap capture folder:  ", dirPath))
						}

						calcFlashEdgeTimes() // These get written to the flashEdgeLogfile
						myWin.flashEdgeLogfile.Close()
						flashEdges = []FlashEdge{}

						clearSchedule(myWin)

						err := MoveFile(myWin.flashEdgeLogfilePath, dirPath+"FLASH_EDGE_TIMES.txt")
						if err != nil {
							log.Println(err)
						}

						_, _ = myWin.logFile.WriteString("Last line of the IotaGFTapp GPS sentence log file" + "\n")

						myWin.logFile.Close()
						err = MoveFile(myWin.logFilePath, dirPath+"IotaGFT_LOG.txt")
						if err != nil {
							log.Println(err)
						}

						// We ignore the error here as it is standard to get a "file used elsewhere" error
						_ = MoveFile(operationLog, dirPath+operationLog)
						//if err != nil {
						//	log.Println(err)
						//}

						// Create a new set of Log and FlashEdge files in our working directory
						createLogAndFlashEdgeFiles(getWorkDir())

						if showTickMsg && needTickMsg {
							fmt.Println(tickMsg)
							needTickMsg = false
						}

						if myWin.autoRunFitsReaderCheckBox.Checked {
							go startFitsReader(dirPath, err)
						}

						if myWin.shutdownCheckBox.Checked {
							if err := exec.Command("cmd", "/C", "shutdown", "/s").Run(); err != nil {
								log.Println("Failed to initiate shutdown:", err)
							}
						}

					}
				endSchedule:
				}
				if showTickMsg && needTickMsg {
					fmt.Println(tickMsg)
				}
			}
			displayEnabledItems(ans, checksumString)
			updateStatusLine(gpsData)

			// Check for selected com port no longer available - an error will occur
			// if the modem status bits cannot be read.  We do this to be as robust as possible
			// to the user disconnecting a device, or adding a device after startup.
			myWin.spMutex.Lock()
			if myWin.serialPort != nil {
				_, err = myWin.serialPort.GetModemStatusBits()
				if err != nil {
					myWin.serialPort = nil
					myWin.comPortName = ""
				}
			}
			myWin.spMutex.Unlock()
		} else {
			time.Sleep(100 * time.Millisecond)
			scanForComPorts()
		}
	}
}

func clearSchedule(myWin *Config) {
	// Reset all scheduling flags
	myWin.utcStartArmed = false
	myWin.pastLeader = false
	myWin.pastFlashOne = false
	myWin.pastFlashTwo = false
	myWin.pastEnd = false

	// Reset the ARm UTC button color and label
	myWin.armUTCbutton.Importance = widget.MediumImportance
	myWin.armUTCbutton.SetText("Arm UTC start")
}

func startFitsReader(dirPath string, err error) {
	cmd := exec.Command("./FitsReader.exe", dirPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
}

func sendSentenceToBeParsed(sentence string, ans []string, err error) ([]string, string, error) {
	n := len(sentence)
	checksum := sentence[n-3:]
	ans, err = parseSentence(sentence[0:n-3], checksum, &gpsData)
	if err != nil {
		addToTextOutDisplay(fmt.Sprintf("%v", err))
		log.Println(fmt.Sprintf("%v", err))
	}
	return ans, checksum, err
}

func updateStatusLine(gpsInfo GPSdata) {
	months := map[string]string{
		"01": "January",
		"02": "February",
		"03": "March",
		"04": "April",
		"05": "May",
		"06": "June",
		"07": "July",
		"08": "August",
		"09": "September",
		"10": "October",
		"11": "November",
		"12": "December",
	}

	if gpsInfo.status != "" {
		myWin.statusStatus.Text = "Status: " + gpsInfo.status
		if strings.Contains(gpsInfo.status, "TimeValid") {
			myWin.statusStatus.Color = color.NRGBA{G: 180, A: 255}
		} else {
			myWin.statusStatus.Color = color.NRGBA{R: 180, A: 255}
		}
		myWin.statusStatus.Refresh()
	} else {
		myWin.statusStatus.Text = "Status: not available"
		myWin.statusStatus.Color = nil
		myWin.statusStatus.Refresh()
	}
	if gpsInfo.date != "" {
		timeStr := fmt.Sprintf("UTC: %s:%s:%s",
			gpsInfo.timeUTC[0:2],
			gpsInfo.timeUTC[2:4],
			gpsInfo.timeUTC[4:6],
		)
		dateStr := fmt.Sprintf("   (%s %s 20%s)",
			gpsInfo.date[0:2],
			months[gpsInfo.date[2:4]],
			gpsInfo.date[4:6],
		)
		myWin.dateTimeStatus.Text = timeStr + dateStr
		myWin.dateTimeStatus.Refresh()
	} else {
		myWin.dateTimeStatus.Text = "Date/time: not available"
		myWin.dateTimeStatus.Refresh()
	}
	if gpsInfo.latitude != "" {
		latText := fmt.Sprintf("Latitude: %s %sd %sm",
			gpsInfo.latDirection,
			gpsInfo.latitude[0:2],
			gpsInfo.latitude[3:],
		)
		myWin.latitudeStatus.Text = latText
		myWin.latitudeStatus.Refresh()
	} else {
		myWin.latitudeStatus.Text = "Latitude: not available"
		myWin.latitudeStatus.Refresh()
	}
	if gpsInfo.longitude != "" {
		lonText := fmt.Sprintf("Longitude: %s %sd %sm",
			gpsInfo.lonDirection,
			gpsInfo.longitude[0:3],
			gpsInfo.longitude[3:],
		)
		myWin.longitudeStatus.Text = lonText
		myWin.longitudeStatus.Refresh()
	} else {
		myWin.longitudeStatus.Text = "Longitude: not available"
		myWin.longitudeStatus.Refresh()
	}
	if gpsInfo.altitude != "" {
		altText := fmt.Sprintf("Altitude: %s %s", gpsInfo.altitude, gpsInfo.altitudeUnits)
		myWin.altitudeStatus.Text = altText
		myWin.altitudeStatus.Refresh()
	} else {
		myWin.altitudeStatus.Text = "Altitude: not available"
		myWin.altitudeStatus.Refresh()
	}
}

func getNextSentence(sc chan string) string {
	// A 'sentence' is everything that precedes a crlf sequence
	started := false // remains false until Arduino emits "[STARTING!]"

	// This is the character sequence that separates 'sentences' from the Arduino
	boundaryMarker := "\r\n"

	// Test code for nested P and E sentences
	sentenceNumber := 0

	// Characters coming in from the serial port arrive in various size
	// 'chunks' that are not on any particular boundary. We accumulate
	// those 'chunks' until a boundaryMarker appears somewhere in sumChunks
	var sumChunks string

	// 'sentence' includes all the characters received up to, but not including
	// a boundary marker.
	var sentence string

	// This sets both storage for and an upper size limit on a 'read chunk'
	buff := make([]byte, 200)

	for { // infinite loop that is never exited
		for { // read chunks loop - may be exited on certain conditions

			myWin.spMutex.Lock()
			if myWin.serialPort == nil { // In case the serialPort is closed, we just do nothing
				myWin.spMutex.Unlock()
				time.Sleep(100 * time.Millisecond)
				//fmt.Println("Found no serial port open")
				break
			}

			// Read a chunk of up to 200 bytes into buff
			n, err := myWin.serialPort.Read(buff)
			if err != nil {
				//log.Print(err)
				myWin.serialPort = nil
			}

			myWin.spMutex.Unlock()

			if n == 0 {
				sc <- "timeout"
				break
			}

			chunk := string(buff[:n])
			sumChunks = sumChunks + chunk

			if strings.Contains(sumChunks, boundaryMarker) {
				sentence, sumChunks, _ = strings.Cut(sumChunks, boundaryMarker)
				if started {
					// Test code for nested P and E sentences
					sentenceNumber += 0 // if 0, then test code is disabled
					if sentenceNumber == 20 {
						//sc <- "{002F{004D92C8 P}*76"
						//sc <- "0AE7 P}*01"
						sc <- "{0033C29E $GPDTM,W84,,{0050BD13 P}*77"
						sc <- "0.0,N,0.0,E,0.0,W84*6F}*3A"
					}
					sc <- sentence
				} else {
					if strings.Contains(sentence, "[STARTING!]") {
						started = true
						sc <- sentence
					}
				}
			}
		} // read chunks loop
	} // infinite loop
}

func displayEnabledItems(ans []string, chkSumStr string) {
	if ans[0] == "" {
		return
	}

	switch ans[0] {
	case "$GPGGA":
		if myWin.gpggaCheckBox.Checked {
			addToTextOutDisplay(ans[1] + chkSumStr)
		}
	case "$GPRMC":
		if myWin.gprmcCheckBox.Checked {
			addToTextOutDisplay(ans[1] + chkSumStr)
		}
	case "$GPDTM":
		if myWin.gpdtmCheckBox.Checked {
			addToTextOutDisplay(ans[1] + chkSumStr)
		}
	case "$PUBX":
		if myWin.pubxCheckBox.Checked {
			addToTextOutDisplay(ans[1] + chkSumStr)
		}
	case "P":
		if myWin.pCheckBox.Checked {
			addToTextOutDisplay(ans[1] + chkSumStr)
		}
	case "MODE":
		if myWin.modeCheckBox.Checked {
			addToTextOutDisplay(ans[1] + chkSumStr)
		}
	default:
		addToTextOutDisplay(ans[1] + chkSumStr)
	}
}

func scanForComPorts() {
	ports, err := getSerialPortsList()
	if err != nil {
		addToTextOutDisplay("Fatal err: could not get list of available com ports")
	}

	var realPorts []string
	for _, port := range ports {
		if port == myWin.comPortName { // Don't fiddle with an open port
			realPorts = append(realPorts, port)

			// But check for duplicate names - duplicate names are generated
			// whenever a com port is disconnected and reconnected (for some unknown reason)
			if !isDuplicate(realPorts, port) {
				realPorts = append(realPorts, port)
			}
			continue
		}
		// Do a 'test open' to see if this is a real serial port
		sp, err := openSerialPort(port, baudrate)
		if err == nil {
			// It's an actual attached and active port
			_ = sp.Close()

			// But check for duplicate names - duplicate names are generated
			// whenever a com port is disconnected and reconnected (for some unknown reason)
			if !isDuplicate(realPorts, port) {
				realPorts = append(realPorts, port)
			}
		}
	}

	// Update the drop-down selection widget
	myWin.portsAvailable = realPorts
	myWin.selectComPort.SetOptions([]string{""})
	myWin.selectComPort.SetOptions(myWin.portsAvailable)

	if len(myWin.portsAvailable) == 0 {
		myWin.selectComPort.ClearSelected()
		myWin.selectComPort.PlaceHolder = "(select one)"
		gpsData = GPSdata{}
	}

	updateStatusLine(gpsData)
	myWin.selectComPort.Refresh()

	if len(myWin.portsAvailable) == 1 {
		myWin.comPortName = myWin.portsAvailable[0]
		myWin.comPortInUse.SetText("Serial port open: " + myWin.portsAvailable[0])
		myWin.selectComPort.SetSelectedIndex(0) // Note: this acts as though the user clicked on this entry
	}
}

func isDuplicate(realPorts []string, port string) bool {
	duplicate := false
	for _, p := range realPorts {
		if p == port {
			duplicate = true
		}
	}
	return duplicate
}
