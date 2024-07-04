import socket, os

MSGLEN = 1000

def makeMsg(msg):
	paddedMsg = msg
	paddedMsg += (MSGLEN - len(msg)) * ' '
	return bytes(paddedMsg.encode("utf"))
	
def sendAutorunTrue():
	HOST = '127.0.0.1'
	PORT = 33001
	with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
		s.connect((HOST, PORT))
		msg = makeMsg('setAutorunTrue')
		s.sendall(msg)
		s.close()
		print('Sent:', 'setAutorunTrue')
 		

sendAutorunTrue()