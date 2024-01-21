package main

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	initializeStartingWindow(&myWin)
	myWin.makeUI()
	os.Exit(m.Run())
}
