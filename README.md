# IotaGFTapp   - an Iota utility

### This utility provides an interface to the IotaGFT device to control it and display its output.

### It also provides a connection to SharpCap (a startup script must be enabled in SharpCap to enable this feature).  Through that connection, IotaGFTapp can start and stop a SharpCap recording.

### When IotaGFTapp is controlling a SharpCap recording, it will automatically insert "goalpost" flash events during the recording, taking into account the exposure time so that at least 10 points will be recorded for the flash.  

### When used to control a SharpCap recording, the recommended work order is:


1) Open the SharpCap application and set (if necessary) the TELESCOP and OBSERVER fields (File|SharpCap settings|Saving).

2) Open the IotaGFTapp and wait for Status to report TimeValid PPS (the GPS chip sometimes need extended time begin emitting GPS time)

3) Select a camera in SharpCap.

4) Any other SharpCap tools and scripts can be run at any convenient time like "plate solve", "goto", etc.

5) Select an exposure time and gain that is appropriate for the observation.

6) Activate the histogram tool in SharpCap.

7) With the LED off, set the black level so that there is no background clipping visible in the histogram.

8) Turn the LED on (the check box at lower right) and use the LED intensity slider to set an intensity that causes
      the histogram to register a small increase in overall pixel intensities. This value is "sticky" between sessions.

9) Set UTC event time (time at center of event) and recording duration and click the Arm UTC start button OR
      leave this field empty, in which case the event time is assumed to be 10 seconds in the future.
      This gives a simple way to run a test recording to make sure that the observation parameters
      are set properly.


NOTE: If you make any changes to the camera exposure time, BE SURE TO click the Arm UTC start button
           again so that the change can be incorporated in the start time and flash duration.

## The IotaGFT device
The IOTA GFT Flash Timer is based on an Arduino Mega2560 R3 with a custom shield.

The shield contains a GPS receiver (with a 1PPS output) and includes a wide dynamic
range LED intensity control to enable the LED flash intensity to be matched to the
camera and telescope configuration while avoiding pixel saturation during flashes.

In operation, the Arduino sends 'sentences' (strings terminated by CR LF). There are
6 'standard' sentences: 4 NMEA GPS sentences plus a P (for pulse) and a MODE that
reports the internal status of the GFT. There are other sentences that will be emitted
in response to 'events' such as a detected error, a command entry, a command response,
the LED turning on or off, etc. These 'other' sentences are always displayed but NOT
written to the standard log.
