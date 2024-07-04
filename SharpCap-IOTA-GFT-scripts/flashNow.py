# This code demonstrates how to send a command to the IotaGFTapp.
# We do not currently expect SharpCap to control the IotaGFTapp which deals better with scheduling
# start of recording and flash timing than SharpCap.
# We leave this script in place for future reference if a use-case can be made for SharpCap
# sending commands to IotaGFTapp

import socket, os

def makeMsg(msg):
	paddedMsg = msg
	paddedMsg += (MSGLEN - len(msg)) * ' '
	return bytes(paddedMsg.encode("utf"))
	
def flashNow():
	HOST = '127.0.0.1'
	PORT = 33001
	with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
		s.connect((HOST, PORT))
		msg = makeMsg('flash now')
		s.sendall(msg)
		s.close()
		print('Sent: flash now')
 		
flashNow()