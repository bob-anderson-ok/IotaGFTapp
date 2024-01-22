package main

import (
	"go.bug.st/serial"
)

func getSerialPortsList() ([]string, error) {
	ports, err := serial.GetPortsList()
	return ports, err
}

func openSerialPort(portName string, baudrate int) (serial.Port, error) {
	mode := &serial.Mode{
		BaudRate: baudrate,
	}
	port, err := serial.Open(portName, mode)
	return port, err
}
