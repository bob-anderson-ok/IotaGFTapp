package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_textOut(t *testing.T) {
	//initialText := "Serial data lines will appear here\nonce the GFT starts up"
	initialText := getInitialText()
	myWin.textOut = getInitialText()
	assert.Equal(t, initialText, myWin.textOut)
}
