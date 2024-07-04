import socket

def makeMsg(msg):
	paddedMsg = msg
	paddedMsg += (MSGLEN - len(msg)) * ' '
	return bytes(paddedMsg.encode("utf"))

def sendFilePath():
	HOST = '127.0.0.1'
	PORT = 33001
	with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
		s.connect((HOST, PORT))
		filePath = SharpCap.GetLastCaptureFilename()
		if not filePath is None:
			msg = makeMsg(filePath)
			s.sendall(msg)
			print('Sent: ', filePath)
		else:
			msg = makeMsg('No filepath available')
			s.sendall(msg)
			print('Sent: ', 'No filepath available')
		s.close()
 		
sendFilePath()