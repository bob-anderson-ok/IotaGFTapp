import socket, os

MSGLEN = 1000

def makeMsg(msg):
	paddedMsg = msg
	paddedMsg += (MSGLEN - len(msg)) * ' '
	return bytes(paddedMsg.encode("utf"))
	
def sendUTCeventStartTime():
	HOST = '127.0.0.1'
	PORT = 33001
	with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
		s.connect((HOST, PORT))
		msg = makeMsg('setUTCeventTime 2026-01-25 10:11:12')
		s.sendall(msg)
		s.close()
		print('Sent:', 'setUTCeventTime 2026-01-25 10:11:12')
 		

sendUTCeventStartTime()