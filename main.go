package main

import (
	"github.com/nsf/termbox-go"
	"log"
)

const (
	ModeNormal int8 = iota
	ModeInsert
	ModeCommand
)

var (
	buffer           = []rune{}
	mode             = ModeNormal // start in normal mode
	commandBuffer    = []rune{}
	running          = true
	cursorX, cursorY int // Track the cursor position
	lineStarts       []int
)

func updateLineStarts() {
	lineStarts = []int{0} // Always start with the first line starting at index 0
	for i, ch := range buffer {
		if ch == '\n' {
			// The next line starts after this newline character
			lineStarts = append(lineStarts, i+1)
		}
	}
	lineStarts = append(lineStarts, len(buffer)) // Add the end of the buffer as the last line start
}

func handleKeyPress(ev termbox.Event) {
	switch mode {
	case ModeNormal:
		if ev.Ch == ':' { // Enter command mode
			mode = ModeCommand
			commandBuffer = []rune{} // Reset command buffer
		} else if ev.Ch == 'i' {
			mode = ModeInsert
		} else if ev.Ch == 'h' {
			moveCursorLeft()
		} else if ev.Ch == 'j' {
			moveCursorDown()
		} else if ev.Ch == 'k' {
			moveCursorUp()
		} else if ev.Ch == 'l' {
			moveCursorRight()
		}
		// TODO: Add more keys here
	case ModeInsert:
		if ev.Key == termbox.KeyEsc {
			mode = ModeNormal
			if cursorX > 0 {
				cursorX--
			}
		} else if ev.Key == termbox.KeySpace {
			buffer = append(buffer, ' ')
			cursorX++
		} else if ev.Key == termbox.KeyEnter {
			insertIndex := lineStarts[cursorY] + cursorX
			if insertIndex > len(buffer) {
				insertIndex = len(buffer)
			}
			buffer = append(buffer[:insertIndex], append([]rune{'\n'}, buffer[insertIndex:]...)...)
			cursorY++
			cursorX = 0
		} else if ev.Key == termbox.KeyBackspace || ev.Key == termbox.KeyBackspace2 {
			if len(buffer) > 0 {
				if cursorX == 0 && cursorY > 0 {
					newlineIndexToRemove := lineStarts[cursorY] - 1
					if newlineIndexToRemove >= 0 && newlineIndexToRemove < len(buffer) {
						// Remove the newline character from the buffer
						buffer = append(buffer[:newlineIndexToRemove], buffer[newlineIndexToRemove+1:]...)

						// Update cursor position
						cursorY--
						cursorX = newlineIndexToRemove - lineStarts[cursorY]
					}
				} else if cursorX > 0 {
					buffer = buffer[:len(buffer)-1]
					cursorX--
				}
			}
		} else if ev.Ch != 0 {
			// Calculate the insertion index
			insertIndex := lineStarts[cursorY] + cursorX
			if insertIndex > len(buffer) {
				insertIndex = len(buffer)
			}
			log.Printf("insertIndex: %v", insertIndex)
			buffer = append(buffer[:insertIndex], append([]rune{ev.Ch}, buffer[insertIndex:]...)...)
			cursorX++
		}
	case ModeCommand:
		if ev.Key == termbox.KeyEnter {
			if string(commandBuffer) == "q" { // Quit command
				termbox.Close()
				exit()
			}
			mode = ModeNormal
		} else if ev.Key == termbox.KeyEsc {
			mode = ModeNormal
		} else if ev.Key == termbox.KeyBackspace || ev.Key == termbox.KeyBackspace2 {
			if len(commandBuffer) > 0 {
				commandBuffer = commandBuffer[:len(commandBuffer)-1]
			}
		} else if ev.Ch != 0 {
			commandBuffer = append(commandBuffer, ev.Ch)
		}
	}
	updateLineStarts()
	render()
}

func moveCursorUp() {
	if cursorY == 0 {
		return
	}
	cursorY--
	length := lineLength(cursorY)
	if cursorX > length {
		cursorX = length - 1
	}
}

func moveCursorDown() {
	if cursorY >= len(lineStarts)-2 {
		return
	}
	cursorY++
	length := lineLength(cursorY)
	log.Printf("length: %v", length)
	if cursorX > length {
		cursorX = length - 1
	}
}

func moveCursorRight() {
	if cursorX+1 < lineLength(cursorY) {
		log.Printf("line: %v", lineLength(cursorY))
		cursorX++
	}
}

func lineLength(line int) int {
	if line > len(lineStarts)-2 {
		return 0
	}
	// Check whether last character of the line is a newline
	if buffer[lineStarts[line+1]-1] == '\n' {
		return lineStarts[line+1] - lineStarts[line] - 1
	}
	return lineStarts[line+1] - lineStarts[line]
}

func moveCursorLeft() {
	if cursorX > 0 {
		cursorX--
	}
}

func render() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	// Display buffer
	// Track positions
	x, y := 0, 0
	w, h := termbox.Size() // Get terminal dimensions

	// Display buffer with support for newlines
	log.Printf("buffer: [%v]", buffer)
	for _, ch := range buffer {
		if ch == '\n' {
			x = 0
			y++
		} else {
			termbox.SetCell(x, y, ch, termbox.ColorDefault, termbox.ColorDefault)
			x++
		}

		if x >= w {
			x = 0
			y++
		}
		if y >= h {
			break // Stop rendering if we run out of space
		}
	}

	// If in command mode, display the command buffer and ':' prompt
	if mode == ModeCommand {
		prompt := ":"
		for i, ch := range prompt {
			termbox.SetCell(i, h-1, ch, termbox.ColorDefault, termbox.ColorDefault) // Display at the bottom
		}
		for i, ch := range commandBuffer {
			termbox.SetCell(i+len(prompt), h-1, ch, termbox.ColorDefault, termbox.ColorDefault) // Display command after the prompt
		}
	} else if mode == ModeInsert {
		// Display "-- INSERT --" at the bottom
		insert := "-- INSERT --"
		for i, ch := range insert {
			termbox.SetCell(i, h-1, ch, termbox.ColorDefault, termbox.ColorDefault)
		}
	}
	log.Printf("Setting cursor at: %v, %v", cursorX, cursorY)
	if cursorX < 0 {
		cursorX = 0
	}
	if cursorY < 0 {
		cursorY = 0
	}
	termbox.SetCursor(cursorX, cursorY)
	termbox.Flush()
}

func exit() {
	running = false
	// Exit the application
	termbox.Close()
}

func main() {
	err := termbox.Init()
	if err != nil {
		log.Fatal(err)
	}
	defer termbox.Close()

	eventQueue := make(chan termbox.Event)
	go func() {
		for {
			eventQueue <- termbox.PollEvent()
		}
	}()

	for running {
		switch ev := <-eventQueue; ev.Type {
		case termbox.EventKey:
			handleKeyPress(ev)
		}
	}
}
