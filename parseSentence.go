package main

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const hexChar = "0123456789ABCDEF"

func parseSentence(sentence string, gpsInfo *GPSdata) ([]string, error) {
	ans := []string{""}
	var deltaP int64

	if strings.Contains(sentence, "$") {
		parts := strings.Split(sentence, " ")
		if len(parts) < 2 {
			return ans, errors.New("parseSentence(): split of $ sentence on space did not give 2 parts")
		}

		// 'payload' removes the leading "{000C97C7 " and the trailing "}". What is left should be standard nmea frame
		// with leading $ and trailing checksum
		payload := parts[1][:len(parts[1])-1]

		if !isChecksumValid(payload) {
			return ans, errors.New("parseSentence(): failed checksum test")
		}

		parts = strings.Split(payload, ",")
		nmeaName := parts[0]
		switch nmeaName {
		case "$GPGGA":
			return []string{"$GPGGA", sentence}, nil
		case "$GPRMC":
			return []string{"$GPRMC", sentence}, nil
		case "$GPDTM":
			return []string{"$GPDTM", sentence}, nil
		case "$PUBX":
			return []string{"$PUBX", sentence}, nil
		default:
			errMsg := fmt.Sprintf("parseSentence(): no decoder enabled for %s", payload)
			return ans, errors.New(errMsg)
		}
	}

	if strings.Contains(sentence, "MODE") {
		parts := strings.Split(sentence, " ")
		if len(parts) < 2 {
			return ans, errors.New("parseSentence(): split of MODE sentence on space did not give 2 parts")
		}
		ans = []string{"MODE", sentence}
		return ans, nil
	}

	if strings.Contains(sentence, "P}") {
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
		if myWin.lastPvalue != 0 {
			deltaP = value - myWin.lastPvalue
			myWin.lastPvalue = value
		} else {
			deltaP = 0
			myWin.lastPvalue = value
		}
		ans = []string{"P", fmt.Sprintf("%s (deltaP is %d)", sentence, deltaP)}
		return ans, nil
	}

	// We reach this point if the sentence type is not recognized.

	return []string{"other", sentence}, nil
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
