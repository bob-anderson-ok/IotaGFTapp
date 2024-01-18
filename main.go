package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"log"
	"os"
)

type Config struct {
	App             fyne.App
	InfoLog         *log.Logger
	ErrorLog        *log.Logger
	MainWindow      fyne.Window
	statusLine      *fyne.Container
	latitudeStatus  *canvas.Text
	longitudeStatus *canvas.Text
	dateTimeStatus  *canvas.Text
	textOut         *widget.Label
}

var myWin Config

func main() {

	initializeStartingWindow(&myWin)

	// create our loggers
	myWin.InfoLog = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	myWin.ErrorLog = log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	myWin.makeUI()

	// show and run the application
	myWin.MainWindow.ShowAndRun()
}

func initializeStartingWindow(myWin *Config) {
	myWin.App = app.New()
	myWin.MainWindow = myWin.App.NewWindow("IOTA GFT app")
	myWin.MainWindow.Resize(fyne.Size{Height: 600, Width: 900})
	myWin.MainWindow.SetMaster() // As 'master', if the window is closed, the application quits.
	myWin.MainWindow.CenterOnScreen()
	myWin.makeUI()
}
