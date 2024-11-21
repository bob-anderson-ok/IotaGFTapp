import socket
import threading

MSGLEN = 1000  # We use fixed size messages to avoid possible tcp 'fragmenting' (delivery of a message in parts)

def makeMsg(msg):
    paddedMsg = msg
    paddedMsg += (MSGLEN - len(msg)) * ' '
    return bytes(paddedMsg.encode("utf"))

def msgTrim(msg):
    return msg.rstrip()

def listeningThread(startedBy):
	# print(startedBy)
	HOST = '127.0.0.1'
	PORT = 33000
	print("IotaGFT SharpCap script version 1.1")
	print("SharpCap is listening on 127.0.0.1:33000" + " (started by: " + startedBy + ")")
	
	with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
		s.bind((HOST, PORT))
		s.listen()
		while True:
			conn, addr = s.accept()
			print(f"Connection established from: {addr}")
			connected = True
			while connected:
				chunks = []
				bytesRcvd = 0
				while bytesRcvd < MSGLEN:
					chunk = conn.recv(min(MSGLEN - bytesRcvd, 1000))
					if not chunk:
						print("Connection lost")
						connected = False
						conn.close()
						break
					else:
						chunks.append(chunk)
						bytesRcvd += len(chunk)
				if not connected:
					break
				data = b"".join(chunks)
				
				message = msgTrim(data.decode("utf-8"))
				print("rcvd message:", message)
				
				if message == "start":
					if not SharpCap.IsCameraSelected:
						conn.sendall(makeMsg("No camera selected"))
					else:
						ok = SharpCap.SelectedCamera.PrepareToCapture()
						if not ok:
							conn.sendall(makeMsg("Capture start failed"))
						else:
							SharpCap.SelectedCamera.RunCapture()
							conn.sendall(makeMsg("OK"))
				
				elif message == "stop":
					if not SharpCap.IsCameraSelected:
						conn.sendall(makeMsg("No camera selected"))
					else:
						SharpCap.SelectedCamera.StopCapture()
						conn.sendall(makeMsg("OK"))
				
				elif message == "lastfilepath":
					lastCaptureFilePath = SharpCap.GetLastCaptureFilename()
					if len(lastCaptureFilePath) > 0:
						print("sent: %s" % SharpCap.GetLastCaptureFilename())
						conn.sendall(makeMsg(SharpCap.GetLastCaptureFilename()))
					else:
						conn.sendall(makeMsg("lastfilepath FAILED"))
				
				elif message == "exposure":
					if not SharpCap.IsCameraSelected:
						conn.sendall(makeMsg("No camera selected"))
					else:
						exposure = SharpCap.SelectedCamera.Controls.Exposure.ExposureMs
						print("Sent:", exposure)
						conn.sendall(makeMsg(f'{exposure}'))
					
				elif message.StartsWith("set_exp_seconds"):
					if not SharpCap.IsCameraSelected:
						conn.sendall(makeMsg("No camera selected"))
					else:
						parts = message.Split()
						if len(parts) < 2:
							print("set_exp_seconds error: no exposure time given")
							conn.sendall(makeMsg("set_exp_seconds error: no exposure time given"))
						else:
							print("Setting exposure to %s seconds" % parts[1])
							newExposure = float(parts[1])
							SharpCap.SelectedCamera.Controls.Exposure.Value = newExposure
							conn.sendall(makeMsg("exposure set to %s seconds" % parts[1]))
				else:
					conn.sendall(makeMsg("invalid command!"))

def main():
	# We start the "listener" as a daemon thread so that it automatically closes (terminates)
	# whenever SharpCap is closed. Without this attribute, the listener process would 
	# become a never-ending background task that could only be killed by 
	# manual intervention using the Task Manager
	
	listener = threading.Thread(target=listeningThread, args=("main",), daemon=True)
	listener.start()
		
main()