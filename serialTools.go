package main

import (
	"go.bug.st/serial"
	"log"
	"time"
)

func getSerialPortsList() ([]string, error) {
	ports, err := serial.GetPortsList()
	return ports, err
}

func openSerialPort(portName string, baudrate int) (serial.Port, error) {
	mode := &serial.Mode{
		BaudRate: baudrate,
	}

	port, err1 := serial.Open(portName, mode)
	if err1 != nil {
		return port, err1
	}

	err2 := port.ResetInputBuffer()
	if err2 != nil {
		log.Println(err2.Error())
		return port, err2
	}

	err3 := port.ResetOutputBuffer()
	if err3 != nil {
		log.Println(err3.Error())
		return port, err3
	}

	err4 := port.SetReadTimeout(2 * time.Second)
	if err4 != nil {
		log.Println(err4.Error())
		return port, err4
	}

	return port, err1
}
