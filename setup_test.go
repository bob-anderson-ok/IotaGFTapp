package main

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	initializeStartingWindow(&myWin)
	os.Exit(m.Run())
}
