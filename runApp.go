package main

import (
	"fmt"
	"image/color"
	"strings"
	"time"
)

func runApp(myWin *Config) {
	sentenceChan := make(chan string, 1)

	// This runs infinitely, sending each sentence received to sentenceChan. It has a 2-second timeout
	// for dealing with a non-responsive serial port and returns "timeout" as a sentence in that case.
	go getNextSentence(sentenceChan)

	waitingForNestFinish := false

	var ans []string
	var err error
	var parts []string
	var partsSaved []string
	var nester, nestee string

	for {
		if myWin.serialPort != nil {
			// A 'sentence' is everything up to, but not including, a crlf sequence.
			// The last three characters of the 'sentence' are a checksum *xx (even for a 'nest')
			// The checksum has not yet been validated at this point.
			sentence := <-sentenceChan // Block until a sentence is returned by go getNextSentence(sentenceChan)

			if sentence == "timeout" {
				addToTextOutDisplay(fmt.Sprintf("Serial port %s is not responding.", myWin.comPortName))
				continue
			}

			// Always write every sentence to the log file
			if myWin.logFile != nil {
				_, fileErr := myWin.logFile.WriteString(sentence + "\n")
				if fileErr != nil {
					fmt.Println(fmt.Errorf("runApp(): %w", fileErr))
				}
			}

			if waitingForNestFinish {
				waitingForNestFinish = false
				nestee = "{" + partsSaved[1] + sentence
				ans, err = sendSentenceToBeParsed(nestee, ans, err)
				displayEnabledItems(ans)
				ans, err = sendSentenceToBeParsed(nester, ans, err)
				displayEnabledItems(ans)
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
			ans, err = sendSentenceToBeParsed(sentence, ans, err)

			updateStatusLine(gpsData)

			displayEnabledItems(ans)

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

func sendSentenceToBeParsed(sentence string, ans []string, err error) ([]string, error) {
	n := len(sentence)
	checksum := sentence[n-3:]
	ans, err = parseSentence(sentence[0:n-3], checksum, &gpsData)
	if err != nil {
		addToTextOutDisplay(fmt.Sprintf("%v", err))
	}
	return ans, err
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

func displayEnabledItems(ans []string) {
	if ans[0] == "" {
		return
	}

	switch ans[0] {
	case "$GPGGA":
		if myWin.gpggaCheckBox.Checked {
			addToTextOutDisplay(ans[1])
		}
	case "$GPRMC":
		if myWin.gprmcCheckBox.Checked {
			addToTextOutDisplay(ans[1])
		}
	case "$GPDTM":
		if myWin.gpdtmCheckBox.Checked {
			addToTextOutDisplay(ans[1])
		}
	case "$PUBX":
		if myWin.pubxCheckBox.Checked {
			addToTextOutDisplay(ans[1])
		}
	case "P":
		if myWin.pCheckBox.Checked {
			addToTextOutDisplay(ans[1])
		}
	case "MODE":
		if myWin.modeCheckBox.Checked {
			addToTextOutDisplay(ans[1])
		}
	default:
		addToTextOutDisplay(ans[1])
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
