package main

import (
	"fmt"
	"log"
	"strings"
	"time"
)

func runApp(myWin *Config) {
	alt := 0
	sentenceChan := make(chan string, 1)
	go getNextSentence(sentenceChan)
	for {
		if myWin.serialPort != nil {
			sentence := <-sentenceChan
			addToTextOutDisplay(sentence)
			//demo(myWin, alt)
			alt += 10

			// Check for selected com port no longer available - an error will occur
			// if the modem status bits cannot be read.
			_, err := myWin.serialPort.GetModemStatusBits()
			if err != nil {
				//addToTextOutDisplay("com port err while getting status bits")
				myWin.serialPort = nil
				//removePortFromSelectList(myWin.comPortName)
				myWin.comPortName = ""
			}
		} else {
			time.Sleep(1 * time.Second)
			scanForComPorts()
			//fmt.Println("myWin.portsAvailable:", myWin.portsAvailable)
		}
	}
}

func getNextSentence(sc chan string) string {
	started := false // remains false until Arduino emits "[STARTING!]"

	// This is the character sequence that separates 'sentences' from the Arduino
	boundaryMarker := "\r\n"

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

			if myWin.serialPort == nil { // In case the serialPort is closed, we just do nothing
				break
			}

			// Read a chunk of up to 200 bytes into buff
			n, err := myWin.serialPort.Read(buff)
			if err != nil {
				log.Print(err)
			}
			if n == 0 {
				fmt.Println("\nEOF")
				break
			}

			chunk := string(buff[:n])
			sumChunks = sumChunks + chunk

			if strings.Contains(sumChunks, boundaryMarker) {
				sentence, sumChunks, _ = strings.Cut(sumChunks, boundaryMarker)
				if started {
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
