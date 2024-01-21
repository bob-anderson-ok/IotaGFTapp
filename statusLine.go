package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

func makeStatusLine(app *Config) *fyne.Container {
	app.latitudeStatus = canvas.NewText("Latitude: not available", nil)
	app.latitudeStatus.Alignment = fyne.TextAlignLeading

	app.longitudeStatus = canvas.NewText("Longitude: not available", nil)
	//app.longitudeStatus.Alignment = fyne.TextAlignTrailing

	app.altitudeStatus = canvas.NewText("Altitude: not available", nil)
	//app.altitudeStatus.Alignment = fyne.TextAlignTrailing

	app.dateTimeStatus = canvas.NewText("UTC date/time: not available", nil)
	//app.dateTimeStatus.Alignment = fyne.TextAlignTrailing

	ans := container.NewGridWithColumns(4,
		app.latitudeStatus, app.longitudeStatus, app.altitudeStatus, app.dateTimeStatus)
	return ans
}
