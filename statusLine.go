package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

func makeStatusLine(app *Config) *fyne.Container {
	//app.statusStatus = canvas.NewText("Status: not available", color.NRGBA{R: 180, A: 255})
	app.statusStatus = canvas.NewText("Status: not available", nil)
	app.statusStatus.Alignment = fyne.TextAlignLeading

	app.latitudeStatus = canvas.NewText("Latitude: not available", nil)

	app.longitudeStatus = canvas.NewText("Longitude: not available", nil)

	app.altitudeStatus = canvas.NewText("Altitude: not available", nil)

	app.dateTimeStatus = canvas.NewText("UTC date/time: not available", nil)

	// 5 items with only 4 columns so that dataTime appears on second line
	ans := container.NewGridWithColumns(4,
		app.statusStatus, app.latitudeStatus, app.longitudeStatus, app.altitudeStatus, app.dateTimeStatus)
	return ans
}
