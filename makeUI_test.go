package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_textOut(t *testing.T) {
	//initialText := "Serial data lines will appear here\nonce the GFT starts up"
	initialText := getInitialText()
	myWin.serDataLines = getInitialText()
	assert.Equal(t, initialText, myWin.serDataLines)
}
