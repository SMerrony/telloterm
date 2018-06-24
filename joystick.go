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
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/SMerrony/tello"
	"github.com/simulatedsimian/joystick"
)

var (
	js       joystick.Joystick
	jsConfig joystickConfig
	err      error
)

const (
	axLeftX = iota
	axLeftY
	axRightX
	axRightY
	axL1
	axL2
	axR1
	axR2
)

const (
	btnX = iota
	btnCircle
	btnTriangle
	btnSquare
	btnL1
	btnL2
	btnL3
	btnR1
	btnR2
	btnR3
	btnUnknown
)

const deadZone = 2000

type joystickConfig struct {
	axes    []int
	buttons []uint
}

var dualShock4Config = joystickConfig{
	axes: []int{
		axLeftX: 0, axLeftY: 1, axRightX: 3, axRightY: 4,
	},
	buttons: []uint{
		btnX: 0, btnCircle: 1, btnTriangle: 2, btnSquare: 3, btnL1: 4,
		btnL2: 6, btnR1: 5, btnR2: 7,
	},
}

var dualShock4ConfigWin = joystickConfig{
	axes: []int{
		axLeftX: 0, axLeftY: 1, axRightX: 2, axRightY: 3,
	},
	buttons: []uint{
		btnX: 1, btnCircle: 2, btnTriangle: 3, btnSquare: 0, btnL1: 4,
		btnL2: 6, btnR1: 5, btnR2: 7,
	},
}

// hotas mapping seems the same on windows and linux
var tflightHotasXConfig = joystickConfig{
	axes: []int{
		axLeftX: 4, axLeftY: 2, axRightX: 0, axRightY: 1,
	},
	buttons: []uint{
		btnR1: 0, btnL1: 1, btnR3: 2, btnL3: 3, btnSquare: 4, btnX: 5,
		btnCircle: 6, btnTriangle: 7, btnR2: 8, btnL2: 9,
	},
}

func printJoystickHelp() {
	fmt.Print(
		`TelloTerm Joystick Control Mapping

Right Stick  Forward/Backward/Left/Right
Left Stick   Up/Down/Turn
Triangle     Takeoff
X            Land
Circle       
Square       Take Photo
L1           Bounce (on/off)
L2           Palm Land
`)
}

func listJoysticks() {
	for jsid := 0; jsid < 10; jsid++ {
		js, err := joystick.Open(jsid)
		if err != nil {
			if jsid == 0 {
				fmt.Println("No joysticks detected")
			}
			return
		}
		fmt.Printf("Joystick ID: %d: Name: %s, Axes: %d, Buttons: %d\n", jsid, js.Name(), js.AxisCount(), js.ButtonCount())
		js.Close()
	}
}

func setupJoystick(id int) bool {
	if jsTypeFlag == nil || *jsTypeFlag == "" {
		log.Fatalln("No joystick type supplied, please use -jstype option")
	}
	js, err = joystick.Open(id)
	if err != nil {
		log.Fatalf("Could not open specified joystick ID:%d\n", id)
	}
	switch *jsTypeFlag {
	case "DualShock4":
		switch runtime.GOOS {
		case "windows":
			jsConfig = dualShock4ConfigWin
		default:
			jsConfig = dualShock4Config
		}
	case "HotasX":
		jsConfig = tflightHotasXConfig
	default:
		log.Fatalf("Unknown joystick type <%s> supplied\n", *jsTypeFlag)
	}
	return true
}

func intAbs(x int16) int16 {
	if x < 0 {
		return -x
	}
	return x
}

func readJoystick(test bool) {
	var (
		sm                 tello.StickMessage
		jsState, prevState joystick.State
		err                error
	)

	for {
		jsState, err = js.Read()

		if err != nil {
			log.Printf("Error reading joystick: %v\n", err)
		}

		sm.Lx = int16(jsState.AxisData[jsConfig.axes[axLeftX]])
		sm.Ly = int16(jsState.AxisData[jsConfig.axes[axLeftY]]) * -1
		sm.Rx = int16(jsState.AxisData[jsConfig.axes[axRightX]])
		sm.Ry = int16(jsState.AxisData[jsConfig.axes[axRightY]]) * -1
		if intAbs(sm.Lx) < deadZone {
			sm.Lx = 0
		}
		if intAbs(sm.Ly) < deadZone {
			sm.Ly = 0
		}
		if intAbs(sm.Rx) < deadZone {
			sm.Rx = 0
		}
		if intAbs(sm.Ry) < deadZone {
			sm.Ry = 0
		}

		if test {
			log.Printf("JS: Lx: %d, Ly: %d, Rx: %d, Ry: %d\n", sm.Lx, sm.Ly, sm.Rx, sm.Ry)
		} else {
			stickChan <- sm

		}

		if jsState.Buttons&(1<<jsConfig.buttons[btnL1]) != 0 && prevState.Buttons&(1<<jsConfig.buttons[btnL1]) == 0 {
			if test {
				log.Println("L1 pressed")
			} else {
				drone.Bounce()
			}

		}
		if jsState.Buttons&(1<<jsConfig.buttons[btnL2]) != 0 && prevState.Buttons&(1<<jsConfig.buttons[btnL2]) == 0 {
			if test {
				log.Println("L2 pressed")
			} else {
				drone.PalmLand()
			}

		}
		if jsState.Buttons&(1<<jsConfig.buttons[btnSquare]) != 0 && prevState.Buttons&(1<<jsConfig.buttons[btnSquare]) == 0 {
			if test {
				log.Println("Square pressed")
			} else {
				drone.TakePicture()
			}

		}
		if jsState.Buttons&(1<<jsConfig.buttons[btnTriangle]) != 0 && prevState.Buttons&(1<<jsConfig.buttons[btnTriangle]) == 0 {
			if test {
				log.Println("Triangle pressed")
			} else {
				drone.TakeOff()
			}

		}
		if jsState.Buttons&(1<<jsConfig.buttons[btnCircle]) != 0 && prevState.Buttons&(1<<jsConfig.buttons[btnCircle]) == 0 {
			if test {
				log.Println("Circle pressed")
			}
		}
		if jsState.Buttons&(1<<jsConfig.buttons[btnX]) != 0 && prevState.Buttons&(1<<jsConfig.buttons[btnX]) == 0 {
			if test {
				log.Println("X pressed")
			} else {
				drone.Land()
			}
		}
		prevState = jsState

		time.Sleep(updatePeriodMs)
	}
}
