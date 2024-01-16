package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"log"
	"os"
)

type Config struct {
	App        fyne.App
	InfoLog    *log.Logger
	ErrorLog   *log.Logger
	MainWindow fyne.Window
	statusLine *fyne.Container
}

func main() {
	var myApp Config

	// create a fyne application
	fyneApp := app.New()
	myApp.App = fyneApp

	// create our loggers
	myApp.InfoLog = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	myApp.ErrorLog = log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	// open a connection to the database

	// create a database repository

	// create and size and center a fyne window
	myApp.MainWindow = fyneApp.NewWindow("IOTA GFT app")
	myApp.MainWindow.Resize(fyne.Size{Height: 600, Width: 900})
	myApp.MainWindow.SetMaster() // As 'master', if the window is closed, the application quits.
	myApp.MainWindow.CenterOnScreen()

	myApp.makeUI()

	// show and run the application
	myApp.MainWindow.ShowAndRun()
}
