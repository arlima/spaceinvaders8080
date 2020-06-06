package main

import (
	"cpu8080"
	"fmt"
	"log"
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

const (
	windowTitle  = "8080 Emulator"
	windowWidth  = 224
	windowHeight = 256
	pitch        = 4 * windowWidth
)

type soundStruct struct {
	wavBuffer []byte
	wavSpec   *sdl.AudioSpec
	deviceID  sdl.AudioDeviceID
}

type machineState struct {
	whichInterrupt uint
	shift0         uint8
	shift1         uint8
	shiftOffset    uint8
	inPort1        uint8
	inPort2        uint8
	lastOutPort3   uint8
	lastOutPort5   uint8
	paused         bool
	screenBuffer   []byte
	sounds         [9]soundStruct
}

func (m *machineState) machineIN(port uint8) uint8 {
	var a uint8
	switch port {
	case 1:
		return m.inPort1
	case 2:
		return m.inPort2
	case 3:
		v := uint16(m.shift1)<<8 | uint16(m.shift0)
		a = uint8((v >> (8 - m.shiftOffset)) & 0xff)
	}
	return a
}

func (m *machineState) loadSound(filename string, position uint8) {
	var err error
	m.sounds[position].wavBuffer, m.sounds[position].wavSpec = sdl.LoadWAV(filename)
	m.sounds[position].deviceID, err = sdl.OpenAudioDevice("", false, m.sounds[position].wavSpec, nil, 0)
	if err != nil {
		log.Fatal(err)
	}
}

func (m *machineState) playSound(position uint8) {
	_ = sdl.QueueAudio(m.sounds[position].deviceID, m.sounds[position].wavBuffer)
	sdl.PauseAudioDevice(m.sounds[position].deviceID, false)
}

func (m *machineState) processSound(port uint8, value uint8) {
	if port == 3 {
		if value != m.lastOutPort3 {
			if (value&0x1) > 0 && (m.lastOutPort3&0x1) == 0 {
				go m.playSound(0)
			}
			if (value&0x2) > 0 && (m.lastOutPort3&0x2) == 0 {
				go m.playSound(1)
			}
			if (value&0x4) > 0 && (m.lastOutPort3&0x4) == 0 {
				go m.playSound(2)
			}
			if (value&0x8) > 0 && (m.lastOutPort3&0x8) == 0 {
				go m.playSound(3)
			}
			m.lastOutPort3 = value
		}
	} else if port == 5 {
		if value != m.lastOutPort5 {
			if (value&0x1) > 0 && (m.lastOutPort5&0x1) == 0 {
				go m.playSound(4)
			}
			if (value&0x2) > 0 && (m.lastOutPort5&0x2) == 0 {
				go m.playSound(5)
			}
			if (value&0x4) > 0 && (m.lastOutPort5&0x4) == 0 {
				go m.playSound(6)
			}
			if (value&0x8) > 0 && (m.lastOutPort5&0x8) == 0 {
				go m.playSound(7)
			}
			if (value&0x10) > 0 && (m.lastOutPort5&0x10) == 0 {
				go m.playSound(8)
			}
			m.lastOutPort5 = value
		}
	}
}

func (m *machineState) machineOUT(port uint8, value uint8) {
	switch port {
	case 2:
		m.shiftOffset = value & 0x7
	case 3:
		m.processSound(port, value)
	case 4:
		m.shift0 = m.shift1
		m.shift1 = value
	case 5:
		m.processSound(port, value)
	}
}

func (m *machineState) setPixel(x int, y int, color byte) {
	index := (y*windowWidth + x) * 4
	m.screenBuffer[index] = color
	m.screenBuffer[index+1] = color
	m.screenBuffer[index+2] = color
}

func machineRun() error {
	var err error
	var window *sdl.Window
	var renderer *sdl.Renderer
	var texture *sdl.Texture

	cpu := cpu8080.CPU8080{}
	cpu.Initialize()
	cpu.RAMLowerLimit = 0
	cpu.RAMUpperLimit = 0xFFFF

	err = sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}
	defer sdl.Quit()

	mState := machineState{}
	mState.whichInterrupt = 1
	mState.inPort2 = 0
	mState.inPort1 = 1 << 3
	mState.screenBuffer = make([]byte, windowWidth*windowHeight*4)

	mState.loadSound("ufo_lowpitch.wav", 0)
	mState.loadSound("shoot.wav", 1)
	mState.loadSound("explosion.wav", 2)
	mState.loadSound("invaderkilled.wav", 3)
	mState.loadSound("fastinvader1.wav", 4)
	mState.loadSound("fastinvader2.wav", 5)
	mState.loadSound("fastinvader3.wav", 6)
	mState.loadSound("fastinvader4.wav", 7)
	mState.loadSound("ufo_highpitch.wav", 8)

	cpu.InputHandler = mState.machineIN
	cpu.OutputHandler = mState.machineOUT

	err = cpu.ReadFileMemory("./ROM/invaders.h", 0)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}
	err = cpu.ReadFileMemory("./ROM/invaders.g", 0x800)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}
	err = cpu.ReadFileMemory("./ROM/invaders.f", 0x1000)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}
	err = cpu.ReadFileMemory("./ROM/invaders.e", 0x1800)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}

	window, err = sdl.CreateWindow(windowTitle, sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		windowWidth*3, windowHeight*3, sdl.WINDOW_SHOWN)
	if err != nil {
		return fmt.Errorf("Failed to create window: %s", err)
	}
	defer window.Destroy()

	renderer, err = sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		return fmt.Errorf("Failed to create renderer: %s", err)
	}
	defer renderer.Destroy()

	texture, err = renderer.CreateTexture(sdl.PIXELFORMAT_RGB888, sdl.TEXTUREACCESS_STREAMING, windowWidth, windowHeight)
	if err != nil {
		return fmt.Errorf("Failed to create texture: %s", err)
	}
	defer renderer.Destroy()

	running := true
	start := time.Now()
	for running {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch et := event.(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.KeyboardEvent:
				if et.Type == sdl.KEYUP {
					switch et.Keysym.Sym {
					case sdl.K_LEFT:
						mState.inPort1 &= 0b11011111 // P1 joystick left
						mState.inPort2 &= 0b11011111 // P2 joystick left
					case sdl.K_RIGHT:
						mState.inPort1 &= 0b10111111 // P1 joystick right
						mState.inPort2 &= 0b10111111 // P2 joystick right
					case sdl.K_SPACE:
						mState.inPort1 &= 0b11101111 // P1 shoot button
						mState.inPort2 &= 0b11101111 // P2 shoot button
					case sdl.K_1:
						mState.inPort1 &= 0b11111011 // P1 start button
					case sdl.K_2:
						mState.inPort1 &= 0b11111101 // P2 start button
					case sdl.K_c:
						mState.inPort1 &= 0b11111110 // coin
					case sdl.K_t:
						mState.inPort2 &= 0b11111011 // Tilt
					}
				} else if et.Type == sdl.KEYDOWN {
					switch et.Keysym.Sym {
					case sdl.K_ESCAPE:
						running = false
					case sdl.K_LEFT:
						mState.inPort1 |= 1 << 5 // P1 joystick left
						mState.inPort2 |= 1 << 5 // P2 joystick left
					case sdl.K_RIGHT:
						mState.inPort1 |= 1 << 6 // P1 joystick right
						mState.inPort2 |= 1 << 6 // P2 joystick right
					case sdl.K_SPACE:
						mState.inPort1 |= 1 << 4 // P1 shoot button
						mState.inPort2 |= 1 << 4 // P2 shoot button
					case sdl.K_1:
						mState.inPort1 |= 1 << 2 // P1 start button
					case sdl.K_2:
						mState.inPort1 |= 1 << 1 // P2 start button
					case sdl.K_c:
						mState.inPort1 |= 1 << 0 // coin
					case sdl.K_t:
						mState.inPort2 |= 1 << 2 // tilt
					}
				}
			}
		}

		t := time.Now()
		elapsed := t.Sub(start).Milliseconds()
		if elapsed > (1000 / 240) {
			if cpu.IntEnable == 1 {
				if mState.whichInterrupt == 1 {
					cpu.GenerateInterrupt(1)
					mState.whichInterrupt = 2
				} else {
					cpu.GenerateInterrupt(2)
					mState.whichInterrupt = 1
				}

				// Draw Screen
				for ny := 0; ny < 224; ny++ {
					for nx := 0; nx < 32; nx++ {
						for b := 0; b < 8; b++ {
							px := ny
							py := windowHeight - (nx*8 + b) - 1
							if cpu.Memory[uint16(0x2400+ny*32+nx)]&(1<<b) > 0 {
								mState.setPixel(px, py, 255)
							} else {
								mState.setPixel(px, py, 0)
							}
						}
					}
				}
				texture.Update(nil, mState.screenBuffer, pitch)
				renderer.Clear()
				renderer.Copy(texture, nil, nil)
				renderer.Present()
			}
			start = time.Now()
		}

		opcode := cpu.Memory[cpu.PC]

		err := cpu.ExecuteOpCode(0)
		if err != nil {
			return fmt.Errorf("%04X : %s \n", opcode, "invalid opcode")
		}
	}
	return nil
}

func main() {
	err := machineRun()
	if err != nil {
		log.Fatal(err)
	}
}
