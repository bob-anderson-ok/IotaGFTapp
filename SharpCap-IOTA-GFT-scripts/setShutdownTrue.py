import socket, os

MSGLEN = 1000

def makeMsg(msg):
	paddedMsg = msg
	paddedMsg += (MSGLEN - len(msg)) * ' '
	return bytes(paddedMsg.encode("utf"))
	
def sendShutdownTrue():
	HOST = '127.0.0.1'
	PORT = 33001
	with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
		s.connect((HOST, PORT))
		msg = makeMsg('setShutdownTrue')
		s.sendall(msg)
		s.close()
		print('Sent:', 'setShutdownTrue')
 		

sendShutdownTrue()