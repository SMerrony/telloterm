// MIT License

// Copyright (c) 2018 Stephen Merrony

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"runtime/pprof"
	"strconv"
	"sync"
	"time"

	"github.com/SMerrony/tello"
	runewidth "github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

const (
	minWidth       = 80
	minHeight      = 24
	updatePeriodMs = 50
	keyPct         = 33 // default speed setting from keyboard control
)

type label struct {
	x, y   int
	fg, bg termbox.Attribute
	text   string
}

var staticLabels = []label{
	label{33, 0, termbox.ColorWhite | termbox.AttrReverse, termbox.ColorDefault, "TelloTerm"},
	label{33, 14, termbox.ColorWhite | termbox.AttrBold, termbox.ColorDefault, "MVO Data"},
	label{33, 17, termbox.ColorWhite | termbox.AttrBold, termbox.ColorDefault, "IMU Data"},
}

type field struct {
	lab    label
	x, y   int
	w      int
	fg, bg termbox.Attribute
	value  string
}

const (
	fHeight = iota
	fBattery
	fWifiStrength
	fMaxHeight
	fLowBattThresh
	fWifiInterference
	fDerivedSpeed
	fGroundSpeed
	fFwdSpeed
	fLatSpeed
	fVertSpeed
	fBattLow
	fBattCrit
	fBattState
	fGroundVis
	fOvertemp
	fLightStrength
	fOnGround
	fHovering
	fFlying
	fFlyMode
	fCameraState
	fDroneFlyTimeLeft
	fDroneBattLeft
	fVelX
	fVelY
	fVelZ
	fPosX
	fPosY
	fPosZ
	fQatW
	fQatX
	fQatY
	fQatZ
	fTemp
	fRoll
	fPitch
	fYaw
	fHome
	fSSID
	fVersion
	fNumFields
)

var fieldsMu sync.RWMutex
var fields [fNumFields]field

func setupFields() {
	fields[fHeight] = field{label{8, 2, termbox.ColorWhite, termbox.ColorDefault, "Height:"}, 16, 2, 5, termbox.ColorWhite, termbox.ColorDefault, "?m"}
	fields[fBattery] = field{label{34, 2, termbox.ColorWhite, termbox.ColorDefault, "Battery:"}, 43, 2, 4, termbox.ColorWhite, termbox.ColorDefault, "?%"}
	fields[fWifiStrength] = field{label{61, 2, termbox.ColorWhite, termbox.ColorDefault, "WiFi:"}, 67, 2, 4, termbox.ColorWhite, termbox.ColorDefault, "?%"}

	fields[fMaxHeight] = field{label{4, 3, termbox.ColorWhite, termbox.ColorDefault, "Max Height:"}, 16, 3, 5, termbox.ColorWhite, termbox.ColorDefault, "?m"}
	fields[fDroneBattLeft] = field{label{34, 3, termbox.ColorWhite, termbox.ColorDefault, "Voltage:"}, 43, 3, 6, termbox.ColorWhite, termbox.ColorDefault, "?"}
	fields[fWifiInterference] = field{label{53, 3, termbox.ColorWhite, termbox.ColorDefault, "Interference:"}, 67, 3, 4, termbox.ColorWhite, termbox.ColorDefault, "?%"}

	fields[fLowBattThresh] = field{label{24, 4, termbox.ColorWhite, termbox.ColorDefault, "Lo Batt Threshold:"}, 43, 4, 4, termbox.ColorWhite, termbox.ColorDefault, "?%"}

	fields[fDerivedSpeed] = field{label{28, 6, termbox.ColorYellow, termbox.ColorDefault, "Derived Speed:"}, 43, 6, 7, termbox.ColorWhite, termbox.ColorDefault, "?m/s"}
	fields[fVertSpeed] = field{label{51, 6, termbox.ColorWhite, termbox.ColorDefault, "Vertical Speed:"}, 67, 6, 7, termbox.ColorWhite, termbox.ColorDefault, "?m/s"}

	fields[fGroundSpeed] = field{label{2, 7, termbox.ColorWhite, termbox.ColorDefault, "Ground Speed:"}, 16, 7, 5, termbox.ColorWhite, termbox.ColorDefault, "?m/s"}
	fields[fFwdSpeed] = field{label{28, 7, termbox.ColorWhite, termbox.ColorDefault, "Forward Speed:"}, 43, 7, 5, termbox.ColorWhite, termbox.ColorDefault, "?m/s"}
	fields[fLatSpeed] = field{label{52, 7, termbox.ColorWhite, termbox.ColorDefault, "Lateral Speed:"}, 67, 7, 5, termbox.ColorWhite, termbox.ColorDefault, "?m/s"}

	fields[fBattLow] = field{label{3, 9, termbox.ColorWhite, termbox.ColorDefault, "Battery Low:"}, 16, 9, 5, termbox.ColorWhite, termbox.ColorDefault, "?"}
	fields[fBattCrit] = field{label{25, 9, termbox.ColorWhite, termbox.ColorDefault, "Battery Critical:"}, 43, 9, 5, termbox.ColorWhite, termbox.ColorDefault, "?"}
	fields[fBattState] = field{label{52, 9, termbox.ColorWhite, termbox.ColorDefault, "Battery State:"}, 67, 9, 5, termbox.ColorWhite, termbox.ColorDefault, "?"}

	fields[fGroundVis] = field{label{1, 10, termbox.ColorWhite, termbox.ColorDefault, "Ground Visual:"}, 16, 10, 5, termbox.ColorWhite, termbox.ColorDefault, "?"}
	fields[fOvertemp] = field{label{25, 10, termbox.ColorWhite, termbox.ColorDefault, "Over Temperature:"}, 43, 10, 5, termbox.ColorWhite, termbox.ColorDefault, "?"}
	fields[fLightStrength] = field{label{51, 10, termbox.ColorWhite, termbox.ColorDefault, "Light Strength:"}, 67, 10, 5, termbox.ColorWhite, termbox.ColorDefault, "?"}

	fields[fOnGround] = field{label{5, 11, termbox.ColorWhite, termbox.ColorDefault, "On Ground:"}, 16, 11, 5, termbox.ColorWhite, termbox.ColorDefault, "?"}
	fields[fHovering] = field{label{33, 11, termbox.ColorWhite, termbox.ColorDefault, "Hovering:"}, 43, 11, 5, termbox.ColorWhite, termbox.ColorDefault, "?"}
	fields[fFlying] = field{label{59, 11, termbox.ColorWhite, termbox.ColorDefault, "Flying:"}, 67, 11, 5, termbox.ColorWhite, termbox.ColorDefault, "?"}

	fields[fCameraState] = field{label{2, 12, termbox.ColorWhite, termbox.ColorDefault, "Camera State:"}, 16, 12, 6, termbox.ColorWhite, termbox.ColorDefault, "?"}
	fields[fFlyMode] = field{label{30, 12, termbox.ColorWhite, termbox.ColorDefault, "Flight Mode:"}, 43, 12, 5, termbox.ColorWhite, termbox.ColorDefault, "?"}
	fields[fDroneFlyTimeLeft] = field{label{49, 12, termbox.ColorWhite, termbox.ColorDefault, "Flight Remaining:"}, 67, 12, 6, termbox.ColorWhite, termbox.ColorDefault, "?"}

	fields[fVelX] = field{label{4, 15, termbox.ColorWhite, termbox.ColorDefault, "X Velocity:"}, 16, 15, 8, termbox.ColorWhite, termbox.ColorDefault, "?"}
	fields[fVelY] = field{label{31, 15, termbox.ColorWhite, termbox.ColorDefault, "Y Velocity:"}, 43, 15, 8, termbox.ColorWhite, termbox.ColorDefault, "?"}
	fields[fVelZ] = field{label{55, 15, termbox.ColorWhite, termbox.ColorDefault, "Z Velocity:"}, 67, 15, 8, termbox.ColorWhite, termbox.ColorDefault, "?"}

	fields[fPosX] = field{label{4, 16, termbox.ColorWhite, termbox.ColorDefault, "X Position:"}, 16, 16, 6, termbox.ColorWhite, termbox.ColorDefault, "?"}
	fields[fPosY] = field{label{31, 16, termbox.ColorWhite, termbox.ColorDefault, "Y Position:"}, 43, 16, 6, termbox.ColorWhite, termbox.ColorDefault, "?"}
	fields[fPosZ] = field{label{55, 16, termbox.ColorWhite, termbox.ColorDefault, "Z Position:"}, 67, 16, 6, termbox.ColorWhite, termbox.ColorDefault, "?"}

	fields[fQatX] = field{label{8, 18, termbox.ColorWhite, termbox.ColorDefault, "X Quat:"}, 16, 18, 6, termbox.ColorWhite, termbox.ColorDefault, "?"}
	fields[fQatY] = field{label{35, 18, termbox.ColorWhite, termbox.ColorDefault, "Y Quat:"}, 43, 18, 6, termbox.ColorWhite, termbox.ColorDefault, "?"}
	fields[fQatZ] = field{label{59, 18, termbox.ColorWhite, termbox.ColorDefault, "Z Quat:"}, 67, 18, 6, termbox.ColorWhite, termbox.ColorDefault, "?"}

	fields[fTemp] = field{label{10, 19, termbox.ColorWhite, termbox.ColorDefault, "Temp:"}, 16, 19, 6, termbox.ColorWhite, termbox.ColorDefault, "?"}
	fields[fQatW] = field{label{35, 19, termbox.ColorWhite, termbox.ColorDefault, "W Quat:"}, 43, 19, 6, termbox.ColorWhite, termbox.ColorDefault, "?"}
	fields[fYaw] = field{label{62, 19, termbox.ColorYellow, termbox.ColorDefault, "Yaw:"}, 67, 19, 6, termbox.ColorWhite, termbox.ColorDefault, "?°"}

	fields[fHome] = field{label{33, 20, termbox.ColorYellow, termbox.ColorDefault, "Home Pos:"}, 43, 20, 5, termbox.ColorWhite, termbox.ColorDefault, "?"}

	fields[fSSID] = field{label{10, 22, termbox.ColorWhite, termbox.ColorDefault, "SSID:"}, 16, 22, 20, termbox.ColorWhite, termbox.ColorDefault, "?"}
	fields[fVersion] = field{label{57, 22, termbox.ColorWhite, termbox.ColorDefault, "Firmware:"}, 67, 22, 10, termbox.ColorWhite, termbox.ColorDefault, "?"}

}

var (
	drone       tello.Tello
	fdLogging   bool
	fdLog       *csv.Writer
	wideVideo   bool
	useJoystick bool
	stickChan   chan<- tello.StickMessage
)

// program flags
var (
	cpuprofile  = flag.String("cpuprofile", "", "Write cpu profile to `file`")
	fdLogFlag   = flag.String("fdlog", "", "Log some CSV flight data to this file")
	joyHelpFlag = flag.Bool("joyhelp", false, "Print help for joystick control mapping and exit")
	jsIDFlag    = flag.Int("jsid", 999, "ID number of joystick to use (see -jslist to get IDs)")
	jsListFlag  = flag.Bool("jslist", false, "List attached joysticks")
	jsTest      = flag.Bool("jstest", false, "Debug joystick mapping")
	jsTypeFlag  = flag.String("jstype", "", "Type of joystick, options are DualShock4, HotasX")
	keyHelpFlag = flag.Bool("keyhelp", false, "Print help for keyboard control mapping and exit")
	x11Flag     = flag.Bool("x11", false, "Use '-vo x11' flag in case mplayer takes over entire window")
)

func main() {
	flag.Parse()
	if *keyHelpFlag {
		printKeyHelp()
		os.Exit(0)
	}
	if *joyHelpFlag {
		printJoystickHelp()
		os.Exit(0)
	}
	if *jsListFlag {
		listJoysticks()
		os.Exit(0)
	}
	if *jsIDFlag != 999 {
		useJoystick = setupJoystick(*jsIDFlag)
	}
	if *jsTest {
		readJoystick(true)
	}
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}
	if *fdLogFlag != "" {
		fdlogFile, err := os.Create(*fdLogFlag)
		if err != nil {
			log.Fatal("Cannot create Flight Log file: ", err)
		}
		defer fdlogFile.Close()
		fdLog = csv.NewWriter(fdlogFile)
		defer fdLog.Flush()
		headers := []string{"Time", "X", "Y", "Z", "Yaw", "FDHeight"}
		err = fdLog.Write(headers)
		if err != nil {
			log.Fatal("Cannot write headers to Flight Log file: ", err)
		}
		fdLogging = true
	}

	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()

	checkTermSize()
	setupFields()
	displayStaticFields()

	displayDataFields() // FIXME remove: testing

	err = drone.ControlConnectDefault()
	if err != nil {
		termbox.Close()
		log.Fatalf("Could not connect to Tello - %v", err)
	}

	// subscribe to FlightData events and ask for regular updates
	fdChan, _ := drone.StreamFlightData(false, updatePeriodMs)
	go func() {
		for {
			tmpFD := <-fdChan
			fieldsMu.Lock()
			updateFields(tmpFD)
			fieldsMu.Unlock()
		}
	}()

	// update data field display regularly
	go func() {
		for {
			displayDataFields()
			time.Sleep(updatePeriodMs * time.Millisecond)
		}
	}()

	// ask for drone data not normally sent
	drone.GetLowBatteryThreshold()
	drone.GetMaxHeight()
	drone.GetSSID()
	drone.GetVersion()

	if useJoystick {
		stickChan, _ = drone.StartStickListener()
		go readJoystick(false)
	}

mainloop:
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			switch ev.Key {
			case termbox.KeyEsc:
				break mainloop
			case termbox.KeyCtrlL:
				termbox.Sync()
				displayStaticFields()
				displayDataFields()
			case termbox.KeySpace:
				drone.Hover()
			case termbox.KeyArrowUp:
				drone.Forward(keyPct)
			case termbox.KeyArrowDown:
				drone.Backward(keyPct)
			case termbox.KeyArrowLeft:
				drone.Left(keyPct)
			case termbox.KeyArrowRight:
				drone.Right(keyPct)
			case termbox.KeyHome:
				if drone.IsHomeSet() {
					drone.AutoFlyToXY(0, 0)
				} else {
					drone.SetHome()
				}
			default:
				switch ev.Ch {
				case 'q':
					break mainloop
				case 'r':
					termbox.Sync()
					displayStaticFields()
					displayDataFields()
				case 'b':
					drone.Bounce()
				case 't':
					drone.TakeOff()
				case 'o':
					drone.ThrowTakeOff()
				case 'l':
					drone.Land()
				case 'p':
					drone.PalmLand()
				case 'w':
					drone.Up(keyPct * 2)
				case 'a':
					drone.TurnLeft(keyPct * 2)
				case 's':
					drone.Down(keyPct * 2)
				case 'd':
					drone.TurnRight(keyPct * 2)
				case 'f':
					drone.TakePicture()
				case 'v':
					startVideo()
				case '0':
					drone.StartSmartVideo(tello.Sv360)
				case '1':
					drone.ForwardFlip()
				case '2':
					drone.BackFlip()
				case '3':
					drone.LeftFlip()
				case '4':
					drone.RightFlip()
				case '+':
					drone.SetFastMode()
				case '-':
					drone.SetSlowMode()
				case '=':
					if wideVideo {
						drone.SetVideoNormal()
					} else {
						drone.SetVideoWide()
					}
					wideVideo = !wideVideo
				}
			}

		}
	}

	if drone.NumPics() > 0 {
		drone.SaveAllPics(fmt.Sprintf("tello_pic_%s", time.Now().Format(time.RFC3339)))
	}
}

func printKeyHelp() {
	fmt.Print(
		`TelloTerm Keyboard Control Mapping

<Cursor Keys> Move Left/Right/Forward/Backward
w|a|s|d       W: Up, S: Down, A: Turn Left, D: Turn Right
<SPACE>       Hover (stop all movement)
<HOME>        Set Home position or fly to Home position
b             Bounce (toggle)
t             Takeoff
o             Throw Takeoff
l             Land
p             Palm Land
0             360 degree smart video flight
1|2|3|4       Flip Fwd/Back/Left/Right
f             Take Picture (Foto)
q/<Escape>    Quit
r/<Ctrl-L>	  Refresh Screen
v             Start Video (mplayer) Window
-             Slow (normal) flight mode
+             Fast (sports) flight mode
=             Switch between normal and wide video mode
`)
}

func tbprint(x, y int, fg, bg termbox.Attribute, msg string) {
	for _, c := range msg {
		termbox.SetCell(x, y, c, fg, bg)
		x += runewidth.RuneWidth(c)
	}
}

func checkTermSize() (w, h int) {
	w, h = termbox.Size()
	if w < minWidth || h < minHeight {
		termbox.Close()
		log.Fatalln("Please resize terminal window to at least 24x80 and restart program.")
	}
	return w, h
}

func displayStaticFields() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	for _, l := range staticLabels {
		tbprint(l.x, l.y, l.fg, l.bg, l.text)
	}
	termbox.Flush()
}

func displayDataFields() {
	fieldsMu.RLock()
	for _, d := range fields {
		tbprint(d.lab.x, d.lab.y, d.lab.fg, d.lab.bg, d.lab.text)
		tbprint(d.x, d.y, d.fg, d.bg, padString(d.value, d.w))
	}
	fieldsMu.RUnlock()
	termbox.Flush()
}

func padString(unpadded string, l int) (padded string) {
	format := "%-" + strconv.Itoa(l) + "v"
	return fmt.Sprintf(format, unpadded)
}

func boolToYN(b bool) string {
	if b {
		return "Y"
	}
	return "N"
}

func updateFields(newFd tello.FlightData) {
	fields[fHeight].value = fmt.Sprintf("%.1fm", float32(newFd.Height)/10)
	fields[fBattery].value = fmt.Sprintf("%d%%", newFd.BatteryPercentage)
	fields[fWifiStrength].value = fmt.Sprintf("%d%%", newFd.WifiStrength)

	fields[fMaxHeight].value = fmt.Sprintf("%dm", newFd.MaxHeight)
	fields[fLowBattThresh].value = fmt.Sprintf("%d%%", newFd.LowBatteryThreshold)
	fields[fWifiInterference].value = fmt.Sprintf("%d%%", newFd.WifiInterference)

	fields[fDerivedSpeed].value = fmt.Sprintf("%.1fm/s", math.Sqrt(float64(newFd.NorthSpeed*newFd.NorthSpeed)+float64(newFd.EastSpeed*newFd.EastSpeed)))
	fields[fGroundSpeed].value = fmt.Sprintf("%dm/s", newFd.GroundSpeed)
	fields[fFwdSpeed].value = fmt.Sprintf("%dm/s", newFd.NorthSpeed)
	fields[fLatSpeed].value = fmt.Sprintf("%dm/s", newFd.EastSpeed)

	fields[fVertSpeed].value = fmt.Sprintf("%dm/s", newFd.VerticalSpeed)

	fields[fBattLow].value = boolToYN(newFd.BatteryLow)
	fields[fBattCrit].value = boolToYN(newFd.BatteryCritical)
	fields[fBattState].value = boolToYN(newFd.BatteryState)

	fields[fGroundVis].value = boolToYN(newFd.DownVisualState)
	fields[fOvertemp].value = boolToYN(newFd.OverTemp)
	fields[fLightStrength].value = fmt.Sprintf("%d", newFd.LightStrength)

	fields[fOnGround].value = boolToYN(newFd.OnGround)
	fields[fHovering].value = boolToYN(newFd.DroneHover)
	fields[fFlying].value = boolToYN(newFd.Flying)

	fields[fFlyMode].value = fmt.Sprintf("%d", newFd.FlyMode)

	fields[fCameraState].value = fmt.Sprintf("%d", newFd.CameraState)
	fields[fDroneFlyTimeLeft].value = fmt.Sprintf("%d", newFd.DroneFlyTimeLeft)
	fields[fDroneBattLeft].value = fmt.Sprintf("%dmV", newFd.BatteryMilliVolts)

	fields[fVelX].value = fmt.Sprintf("%dcm/s", newFd.MVO.VelocityX)
	fields[fVelY].value = fmt.Sprintf("%dcm/s", newFd.MVO.VelocityY)
	fields[fVelZ].value = fmt.Sprintf("%dcm/s", newFd.MVO.VelocityZ)

	fields[fPosX].value = fmt.Sprintf("%f", newFd.MVO.PositionX)
	fields[fPosY].value = fmt.Sprintf("%f", newFd.MVO.PositionY)
	fields[fPosZ].value = fmt.Sprintf("%f", newFd.MVO.PositionZ)

	fields[fQatW].value = fmt.Sprintf("%f", newFd.IMU.QuaternionW)
	fields[fQatX].value = fmt.Sprintf("%f", newFd.IMU.QuaternionX)
	fields[fQatY].value = fmt.Sprintf("%f", newFd.IMU.QuaternionY)
	fields[fQatZ].value = fmt.Sprintf("%f", newFd.IMU.QuaternionZ)
	fields[fTemp].value = fmt.Sprintf("%dC", newFd.IMU.Temperature)

	// p, r, y := tello.QuatToEulerDeg(newFd.IMU.QuaternionX, newFd.IMU.QuaternionY, newFd.IMU.QuaternionZ, newFd.IMU.QuaternionW)
	// fields[fRoll].value = fmt.Sprintf("%d", r)
	// fields[fPitch].value = fmt.Sprintf("%d", p)
	fields[fYaw].value = fmt.Sprintf("%d°", newFd.IMU.Yaw)

	if drone.IsHomeSet() {
		fields[fHome].value = "Unset"
	} else {
		fields[fHome].value = "Set"
	}

	fields[fSSID].value = newFd.SSID
	fields[fVersion].value = newFd.Version

	if fdLogging {
		logLine := []string{time.Now().Format("15:04:05.000"), fmt.Sprintf("%f", newFd.MVO.PositionX),
			fmt.Sprintf("%f", newFd.MVO.PositionY), fmt.Sprintf("%f", newFd.MVO.PositionZ),
			fmt.Sprintf("%d", newFd.IMU.Yaw), fmt.Sprintf("%.1f", float32(newFd.Height)/10)}
		fdLog.Write(logLine)
	}
}

func startVideo() {
	videochan, err := drone.VideoConnectDefault()
	if err != nil {
		log.Fatalf("Tello VideoConnectDefault() failed with error %v", err)
	}

	// start external mplayer instance...
	// the -vo X11 parm allows it to run nicely inside a virtual machine
	// setting the FPS to 60 seems to produce smoother video
	var player *exec.Cmd
	if *x11Flag {
		player = exec.Command("mplayer", "-nosound", "-vo", "x11", "-fps", "60", "-")
	} else {
		player = exec.Command("mplayer", "-nosound", "-fps", "60", "-")
	}

	playerIn, err := player.StdinPipe()
	if err != nil {
		log.Fatalf("Unable to get STDIN for mplayer %v", err)
	}
	if err := player.Start(); err != nil {
		log.Fatalf("Unable to start mplayer - %v", err)
		return
	}

	// start video feed when drone connects
	drone.StartVideo()
	go func() {
		for {
			drone.StartVideo()
			time.Sleep(500 * time.Millisecond)
		}
	}()

	go func() {
		for {
			vbuf := <-videochan
			_, err := playerIn.Write(vbuf)
			if err != nil {
				log.Fatalf("Error writing to mplayer %v\n", err)
			}
		}
	}()
}
