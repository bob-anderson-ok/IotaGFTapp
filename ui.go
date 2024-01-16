package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

func (app *Config) makeUI() {

	// Make statusLine
	latitude := canvas.NewText("Latitude", nil)
	latitude.Alignment = fyne.TextAlignLeading
	longitude := canvas.NewText("Longitude", nil)
	longitude.Alignment = fyne.TextAlignCenter
	dateTime := canvas.NewText("date and time    ", nil)
	dateTime.Alignment = fyne.TextAlignTrailing
	statusLine := container.NewGridWithColumns(3,
		latitude, longitude, dateTime)

	app.statusLine = statusLine

	finalContent := container.NewVBox(statusLine)
	app.MainWindow.SetContent(finalContent)
}
