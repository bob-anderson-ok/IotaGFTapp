package main

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const hexChar = "0123456789ABCDEF"

func calcChecksum(str string) (string, uint8) {
	checksum := uint8(0)
	for _, c := range str {
		char := uint8(c)
		checksum = checksum ^ char
	}
	return fmt.Sprintf("*%02X", checksum), checksum
}

func parseSentence(sentence, checksum string, gpsInfo *GPSdata) ([]string, error) {
	ans := []string{""}
	var deltaP int64

	chkSum, _ := calcChecksum(sentence)

	if chkSum != checksum {
		fmt.Printf("%s  %s\n", checksum, chkSum)
		return ans, errors.New("parseSentence() found bad checksum in " + sentence)
	}

	// Test for an embedded command response in the sentence (assumed to be an NMEA)
	parts := strings.Split(sentence, "[]")
	if len(parts) == 3 {
		// We have an embedded command response. We simply remove it and print it out.
		sentence = parts[0] + parts[2]
		embeddedCommandResponse := parts[1]
		fmt.Printf("embedded command response: %s\n", embeddedCommandResponse)
	}

	if strings.Contains(sentence, "$") { // process nmea sentence
		parts := strings.Split(sentence, " ")
		if len(parts) < 2 {
			return ans, errors.New("parseSentence(): split of $ sentence on space did not give 2 parts")
		}

		// 'payload' removes the leading "{000C97C7 " and the trailing "}". What is left should be standard nmea frame
		// with leading $ and trailing checksum
		//payload := parts[1][:len(parts[1])-1]
		payload := removeTrailingCharacter(parts[1])

		if !isChecksumValid(payload) {
			return ans, errors.New("parseSentence() found bad checksum in " + payload)
		}

		parts = strings.Split(payload, ",")
		nmeaName := parts[0]
		switch nmeaName {
		case "$GPGGA":
			gpsInfo.altitude = parts[9]
			gpsInfo.altitudeUnits = parts[10]
			return []string{"$GPGGA", sentence}, nil
		case "$GPRMC":
			gpsInfo.timeUTC = parts[1]

			gpsInfo.latitude = parts[3]
			gpsInfo.latDirection = parts[4]
			gpsInfo.longitude = parts[5]
			gpsInfo.lonDirection = parts[6]
			gpsInfo.date = parts[9]

			if len(gpsInfo.timeUTC) < 2 {
				return []string{"$GPRMC", sentence}, nil
			} else {
				gpsInfo.hour, _ = strconv.Atoi(gpsInfo.timeUTC[0:2])
				gpsInfo.minute, _ = strconv.Atoi(gpsInfo.timeUTC[2:4])
				gpsInfo.second, _ = strconv.Atoi(gpsInfo.timeUTC[4:6])
				gpsInfo.year, _ = strconv.Atoi(gpsInfo.date[4:6])
				gpsInfo.year += 2000
				gpsInfo.month, _ = strconv.Atoi(gpsInfo.date[2:4])
				gpsInfo.day, _ = strconv.Atoi(gpsInfo.date[0:2])
				unixTime := time.Date(gpsInfo.year, time.Month(gpsInfo.month), gpsInfo.day,
					gpsInfo.hour, gpsInfo.minute, gpsInfo.second, 0, time.UTC).Unix()
				if gpsInfo.unixTime == 0 {
					gpsInfo.nextUnixTime = unixTime + 1
					if myWin.App.Preferences().BoolWithFallback("ArmUTCstartTime", false) {
						fmt.Println("We need to click the Arm UTC button")
						gpsInfo.unixTime = gpsInfo.nextUnixTime
						armUTCstart()
					}
				}
				gpsInfo.unixTime = unixTime + 1
			}
			return []string{"$GPRMC", sentence}, nil
		case "$GPDTM":
			return []string{"$GPDTM", sentence}, nil
		case "$PUBX":
			gpsInfo.gpsUtcOffset = parts[6]
			if gpsInfo.date != "" {
				calcGPSfromUTC(gpsInfo)
			}
			return []string{"$PUBX", sentence}, nil
		default:
			errMsg := fmt.Sprintf("parseSentence(): no decoder enabled for %s", payload)
			return ans, errors.New(errMsg)
		}
	}

	if strings.Contains(sentence, "MODE") { // process mode sentence
		parts := strings.Split(sentence, " ")
		if len(parts) < 2 {
			return ans, errors.New("parseSentence(): split of MODE sentence on space did not give 2 parts")
		}
		gpsInfo.status = sentence[6 : len(sentence)-1]
		ans = []string{"MODE", sentence}
		return ans, nil
	}

	pSentence := strings.Contains(sentence, "P}") ||
		strings.Contains(sentence, "E}") ||
		strings.Contains(sentence, "+}") ||
		strings.Contains(sentence, "!}")

	if pSentence { // process P sentence
		tickPulse := strings.Contains(sentence, "P}")
		pType := "P"
		parts := strings.Split(sentence, " ")
		if len(parts) < 2 {
			return ans, errors.New("parseSentence(): split of P sentence on space did not give 2 parts")
		}

		// We want to return the P count, not as a hex string, but as an integer string, so we convert it here.
		hexStr := parts[0][1:]
		value, err := strconv.ParseInt(hexStr, 16, 64)
		if err != nil {
			return ans, errors.New("parseSentence(): hex conversion of P sentence value failed")
		}

		// Extract the micro tick time of the current pulse
		if myWin.gotFirst1PPS { // We're past the initial P sentence
			if value > myWin.lastPvalue {
				deltaP = value - myWin.lastPvalue
			} else {
				deltaP = 0xffffffff - myWin.lastPvalue + value + 1
			}
			myWin.lastPvalue = value

			onePPSdata.runningTickTime += deltaP
			onePPSdata.pDelta = append(onePPSdata.pDelta, deltaP)
			if tickPulse {
				newTickStamp := TickStamp{
					utcTime:         gpsInfo.utcTimestamp,
					gpsTime:         gpsInfo.gpsTimestamp,
					runningTickTime: onePPSdata.runningTickTime,
					tickTime:        0,
				}
				onePPSdata.tickStamp = append(onePPSdata.tickStamp, newTickStamp)
			}

		} else { // This is the first P sentence received - initialize onePPSdata structure
			// It is possible at startup that 1pps occurs before the nmea sentence with time info.
			// We just skip that one.
			if gpsInfo.utcTimestamp != "" {
				myWin.gotFirst1PPS = true
				onePPSdata.startTime = gpsInfo.utcTimestamp
				deltaP = 0
				onePPSdata.pDelta = append(onePPSdata.pDelta, deltaP)
				onePPSdata.runningTickTime = value
				myWin.lastPvalue = value
				if tickPulse {
					newTickStamp := TickStamp{
						utcTime:         gpsInfo.utcTimestamp,
						gpsTime:         gpsInfo.gpsTimestamp,
						runningTickTime: onePPSdata.runningTickTime,
						tickTime:        0,
					}
					onePPSdata.tickStamp = append(onePPSdata.tickStamp, newTickStamp)
				}
			}
		}

		// The position of this code is important - it must follow the extraction of micro tick
		// time from a P sentence so that onePPSdata.runningTickTime has been updated
		if strings.Contains(sentence, "+}") { // process flashOn sentence
			//fmt.Printf("Flash on  @ %s  %s\n", sentence, gpsInfo.utcTimestamp)
			pType = "+"
			if myWin.pastLeader {
				flashEdges = append(flashEdges, FlashEdge{
					edgeTime: onePPSdata.runningTickTime,
					on:       true,
				})
			}
		}

		// The position of this code is important - it must follow the extraction of micro tick
		// time from a P sentence so that onePPSdata.runningTickTime has been updated
		if strings.Contains(sentence, "!}") { // process flashOff sentence
			//fmt.Printf("Flash off @ %s  %s\n", sentence, gpsInfo.utcTimestamp)
			pType = "+"
			if myWin.pastLeader {
				flashEdges = append(flashEdges, FlashEdge{
					edgeTime: onePPSdata.runningTickTime,
					on:       false,
				})
			}
		}
		if strings.Contains(sentence, "E}") {
			pType = "E"
		}
		ans = []string{pType, fmt.Sprintf("%s (deltaP is %d)", sentence, deltaP)}
		return ans, nil
	}

	// We reach this point if the sentence type is not one we need to process
	// That includes {ERROR ...}  [CMD ...] and [ ... ] (which are command responses

	return []string{"other", sentence}, nil
}

func convertTimestampToTimeObject(ts string) (time.Time, error) {
	location, _ := time.LoadLocation("") // specify UTC
	year, err := strconv.Atoi(ts[0 : 3+1])
	if err != nil {
		return time.Time{}, err
	}

	month, err := strconv.Atoi(ts[5 : 6+1])
	if err != nil {
		return time.Time{}, err
	}

	day, err := strconv.Atoi(ts[8 : 9+1])
	if err != nil {
		return time.Time{}, err
	}

	hour, err := strconv.Atoi(ts[11 : 12+1])
	if err != nil {
		return time.Time{}, err
	}

	minute, err := strconv.Atoi(ts[14 : 15+1])
	if err != nil {
		return time.Time{}, err
	}

	second, err := strconv.Atoi(ts[17 : 18+1])
	if err != nil {
		return time.Time{}, err
	}
	timeObject := time.Date(
		year,
		time.Month(month),
		day,
		hour,
		minute,
		second,
		0,
		location)
	return timeObject, nil
}

func calcDeltaSeconds(earlyTime, lateTime string) int64 {
	timeEarly, err := convertTimestampToTimeObject(earlyTime)
	if err != nil {
		panic(err)
	}
	timeLate, err := convertTimestampToTimeObject(lateTime)
	if err != nil {
		panic(err)
	}
	bob := timeLate.Sub(timeEarly)
	return int64(bob.Seconds())
}

func calcAdderToTimestamp(ts string, addedTime float64) string {
	tsTimeObject, err := convertTimestampToTimeObject(ts)
	if err != nil {
		panic(err)
	}
	microsecondsToAdd := time.Duration(addedTime*1_000_000+0.5) * time.Microsecond
	augmentedTime := tsTimeObject.Add(microsecondsToAdd)
	return convertTimeObjectToTimestamp(augmentedTime)
}

func convertTimeObjectToTimestamp(t time.Time) string {
	newTimestamp := fmt.Sprintf("%4d-%02d-%02dT%02d:%02d:%02d.%06d",
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond()/1000)
	return newTimestamp
}

func calcGPSfromUTC(g *GPSdata) {

	location, _ := time.LoadLocation("") // specify UTC
	utcTime := time.Date(
		g.year,
		time.Month(g.month),
		g.day,
		g.hour,
		g.minute,
		g.second,
		0,
		location)

	// Deal with possibility that the gpsUtcOffset ends in D (default offset
	var cleanOffset string
	cleanOffset = strings.Replace(g.gpsUtcOffset, "D", "", 1)
	gpsOffset, _ := strconv.Atoi(cleanOffset)
	gpsTime := utcTime.Add(-time.Duration(gpsOffset) * time.Second)
	g.utcTimestamp = fmt.Sprintf("%4d-%02d-%02dT%02d:%02d:%02d.000000",
		g.year, g.month, g.day, g.hour, g.minute, g.second)
	g.gpsTimestamp = fmt.Sprintf("%4d-%02d-%02dT%02d:%02d:%02d.000000",
		gpsTime.Year(), gpsTime.Month(), gpsTime.Day(), gpsTime.Hour(), gpsTime.Minute(), gpsTime.Second())
}

func isChecksumValid(frameAsString string) bool {
	frame := []byte(frameAsString)

	start, end := bytes.IndexByte(frame, '$'), bytes.LastIndexByte(frame, '*')
	if start == -1 || end == -1 || end+3 > len(frame) {
		return false
	}

	var x byte
	for _, v := range frame[start+1 : end] {
		x ^= v
	}
	chk := strings.ToUpper(string(frame[end+1 : end+3]))
	if chk[0] != hexChar[x>>4] || chk[1] != hexChar[x&0xf] { // also lowercase?
		return false
	}
	return true
}

func removeTrailingCharacter(part string) string {
	return part[:len(part)-1]
}
