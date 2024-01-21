package main

import (
	"fmt"
	"time"
)

func runApp(myWin *Config) {

	alt := 0
	for {
		time.Sleep(2500 * time.Millisecond)
		newAltitudeMsg := fmt.Sprintf("Altitude: %d meters", alt)
		myWin.altitudeStatus.Text = newAltitudeMsg
		myWin.altitudeStatus.Refresh()
		alt += 10

		myWin.serDataLines = append(myWin.serDataLines, newAltitudeMsg)
		if len(myWin.serDataLines) > MaxSerialDataLines {
			fmt.Println("Max serial data reached. Clearing all.")
			myWin.serDataLines = []string{""}
		}
		if myWin.autoScroll.Checked {
			myWin.serOutList.ScrollToBottom()
		}
	}
}
