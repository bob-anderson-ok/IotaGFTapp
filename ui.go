package main

import (
	"fmt"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (app *Config) makeUI() {

	app.statusLine = makeStatusLine(app)
	finalContent := container.NewVBox(app.statusLine)

	app.textOut = widget.NewLabel(getInitialText(10))

	ctr := container.NewVScroll(app.textOut)
	finalContent.Add(ctr)

	app.MainWindow.SetContent(finalContent)
}

func getInitialText(numLines int) string {
	var ans, newLine string
	for i := 0; i < numLines; i++ {
		newLine = fmt.Sprintf("%d\n", i)
		ans += newLine
	}
	return ans
}
