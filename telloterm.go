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
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
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
)

type label struct {
	x, y   int
	fg, bg termbox.Attribute
	text   string
}

var staticLabels = []label{
	label{32, 0, termbox.ColorWhite | termbox.AttrReverse, termbox.ColorDefault, "TelloTerm"},

	label{2, 22, termbox.ColorWhite, termbox.ColorDefault, "SSID: "},
	label{42, 22, termbox.ColorWhite, termbox.ColorDefault, "Firmware: "},
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
	fBattLow
	fBattCrit
	fBattState
	fGroundVis
	fOvertemp
	fLightStrength
	fOnGround
	fHovering
	fFlying
	fCameraState
	fDroneFlyTimeLeft
	fDroneBattLeft
	fNumFields
)

var fieldsMu sync.RWMutex
var fields [fNumFields]field

func setupFields() {
	fields[fHeight] = field{label{8, 2, termbox.ColorWhite, termbox.ColorDefault, "Height: "}, 16, 2, 5, termbox.ColorWhite, termbox.ColorDefault, "0.0m"}
	fields[fBattery] = field{label{33, 2, termbox.ColorWhite, termbox.ColorDefault, "Battery: "}, 42, 2, 4, termbox.ColorWhite, termbox.ColorDefault, "100%"}
	fields[fWifiStrength] = field{label{60, 2, termbox.ColorWhite, termbox.ColorDefault, "WiFi: "}, 66, 2, 4, termbox.ColorWhite, termbox.ColorDefault, "100%"}

	fields[fMaxHeight] = field{label{4, 4, termbox.ColorWhite, termbox.ColorDefault, "Max Height: "}, 16, 4, 5, termbox.ColorWhite, termbox.ColorDefault, "0.0m"}
	fields[fLowBattThresh] = field{label{23, 4, termbox.ColorWhite, termbox.ColorDefault, "Lo Batt Threshold: "}, 42, 4, 4, termbox.ColorWhite, termbox.ColorDefault, "100%"}
	fields[fWifiInterference] = field{label{52, 4, termbox.ColorWhite, termbox.ColorDefault, "Interference: "}, 66, 4, 4, termbox.ColorWhite, termbox.ColorDefault, "100%"}

	fields[fDerivedSpeed] = field{label{27, 6, termbox.ColorYellow, termbox.ColorDefault, "Derived Speed: "}, 42, 6, 6, termbox.ColorWhite, termbox.ColorDefault, "0m/s"}
	fields[fGroundSpeed] = field{label{2, 7, termbox.ColorWhite, termbox.ColorDefault, "Ground Speed: "}, 16, 7, 5, termbox.ColorWhite, termbox.ColorDefault, "0m/s"}
	fields[fFwdSpeed] = field{label{27, 7, termbox.ColorWhite, termbox.ColorDefault, "Forward Speed: "}, 42, 7, 5, termbox.ColorWhite, termbox.ColorDefault, "0m/s"}
	fields[fLatSpeed] = field{label{51, 7, termbox.ColorWhite, termbox.ColorDefault, "Lateral Speed: "}, 66, 7, 5, termbox.ColorWhite, termbox.ColorDefault, "0m/s"}

	fields[fBattLow] = field{label{3, 9, termbox.ColorWhite, termbox.ColorDefault, "Battery Low: "}, 16, 9, 5, termbox.ColorWhite, termbox.ColorDefault, "N"}
	fields[fBattCrit] = field{label{24, 9, termbox.ColorWhite, termbox.ColorDefault, "Battery Critical: "}, 42, 9, 5, termbox.ColorWhite, termbox.ColorDefault, "N"}
	fields[fBattState] = field{label{51, 9, termbox.ColorWhite, termbox.ColorDefault, "Battery State: "}, 66, 9, 5, termbox.ColorWhite, termbox.ColorDefault, "N"}

	fields[fGroundVis] = field{label{1, 10, termbox.ColorWhite, termbox.ColorDefault, "Ground Visual: "}, 16, 10, 5, termbox.ColorWhite, termbox.ColorDefault, "N"}
	fields[fOvertemp] = field{label{24, 10, termbox.ColorWhite, termbox.ColorDefault, "Over Temperature: "}, 42, 10, 5, termbox.ColorWhite, termbox.ColorDefault, "N"}
	fields[fLightStrength] = field{label{50, 10, termbox.ColorWhite, termbox.ColorDefault, "Light Strength: "}, 66, 10, 5, termbox.ColorWhite, termbox.ColorDefault, "0"}

	fields[fOnGround] = field{label{5, 11, termbox.ColorWhite, termbox.ColorDefault, "On Ground: "}, 16, 11, 5, termbox.ColorWhite, termbox.ColorDefault, "N"}
	fields[fHovering] = field{label{32, 11, termbox.ColorWhite, termbox.ColorDefault, "Hovering: "}, 42, 11, 5, termbox.ColorWhite, termbox.ColorDefault, "N"}
	fields[fFlying] = field{label{58, 11, termbox.ColorWhite, termbox.ColorDefault, "Flying: "}, 66, 11, 5, termbox.ColorWhite, termbox.ColorDefault, "N"}

	fields[fCameraState] = field{label{2, 12, termbox.ColorWhite, termbox.ColorDefault, "Camera State:"}, 16, 12, 6, termbox.ColorWhite, termbox.ColorDefault, "?"}
	fields[fDroneFlyTimeLeft] = field{label{32, 12, termbox.ColorWhite, termbox.ColorDefault, "Flight Remaining:"}, 42, 12, 6, termbox.ColorWhite, termbox.ColorDefault, "?"}
	fields[fDroneBattLeft] = field{label{58, 12, termbox.ColorWhite, termbox.ColorDefault, "Drone Batt:"}, 66, 12, 6, termbox.ColorWhite, termbox.ColorDefault, "?"}

}

var (
	drone     tello.Tello
	wideVideo bool
)

// program flags
var (
	x11Flag = flag.Bool("x11", false, "Use '-vo x11' flag in case mplayer takes over entire window")
	//joyHelpFlag = flag.Bool("joyhelp", false, "Print help for joystick control mapping and exit")
	keyHelpFlag = flag.Bool("keyhelp", false, "Print help for keyboard control mapping and exit")
)

func main() {
	flag.Parse()
	if *keyHelpFlag {
		printKeyHelp()
		os.Exit(0)
	}
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()

	//w, h := checkTermSize()
	checkTermSize()

	setupFields()

	displayStaticFields()

	// updateFields(flightData) // FIXME remove:testing
	// displayDataFields()      // FIXME remove: testing

	err = drone.ControlConnectDefault()
	if err != nil {
		termbox.Close()
		log.Fatalf("Could not connect to Tello - %v", err)
	}

	// subscribe to FlightData events and ask for regularupdates
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
				drone.Forward(25)
			case termbox.KeyArrowDown:
				drone.Backward(25)
			case termbox.KeyArrowLeft:
				drone.Left(25)
			case termbox.KeyArrowRight:
				drone.Right(25)
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
					drone.Up(50)
				case 'a':
					drone.TurnLeft(50)
				case 's':
					drone.Down(50)
				case 'd':
					drone.TurnRight(50)
				case 'f':
					drone.TakePicture()
				case 'v':
					startVideo()
				case '1':
					drone.ForwardFlip()
				case '2':
					drone.BackFlip()
				case '3':
					drone.LeftFlip()
				case '4':
					drone.RightFlip()
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
b             Bounce (toggle)
t             Takeoff
o             Throw Takeoff
l             Land
p             Palm Land
1|2|3|4       Flip Fwd/Back/Left/Right
f             Take Picture (Foto)
q/<Escape>    Quit
r/<Ctrl-L>	  Refresh Screen
v             Start Video (mplayer) Window
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

	fields[fBattLow].value = boolToYN(newFd.BatteryLow)
	fields[fBattCrit].value = boolToYN(newFd.BatteryCritical)
	fields[fBattState].value = boolToYN(newFd.BatteryState)

	fields[fGroundVis].value = boolToYN(newFd.DownVisualState)
	fields[fOvertemp].value = boolToYN(newFd.OverTemp)
	fields[fLightStrength].value = fmt.Sprintf("%d", newFd.LightStrength)

	fields[fOnGround].value = boolToYN(newFd.OnGround)
	fields[fHovering].value = boolToYN(newFd.DroneHover)
	fields[fFlying].value = boolToYN(newFd.Flying)

	fields[fCameraState].value = fmt.Sprintf("%d", newFd.CameraState)
	fields[fDroneFlyTimeLeft].value = fmt.Sprintf("%d", newFd.DroneFlyTimeLeft)
	fields[fDroneBattLeft].value = fmt.Sprintf("%dmV", newFd.BatteryMilliVolts)

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
			time.Sleep(time.Second)
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
