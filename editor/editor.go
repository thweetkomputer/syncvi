package editor

import (
	"context"
	"github.com/nsf/termbox-go"
	"log"
	"os"
	"strconv"
	raftpb "syncvi/raft/rpc"
	"syncvi/storage"
)

const (
	ModeNormal int8 = iota
	ModeInsert
	ModeCommand
)

var (
	buffer                   = []rune{}
	mode                     = ModeNormal // start in normal mode
	commandBuffer            = []rune{}
	running                  = true
	cursorX, cursorY         int // Track the cursor position
	viewOffsetX, viewOffsetY int // Track the view offset
	lineStarts               []int
	filePath                 string
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
	if cursorX+viewOffsetX > lineLength(cursorY+viewOffsetY) {
		cursorX = lineLength(cursorY + viewOffsetY)
	}
}

func valid(ev termbox.Event) bool {
	if ev.Key == termbox.KeyEsc ||
		ev.Key == termbox.KeyBackspace ||
		ev.Key == termbox.KeyBackspace2 ||
		ev.Key == termbox.KeySpace ||
		ev.Key == termbox.KeyEnter {
		return true
	}
	if ev.Ch >= 32 && ev.Ch <= 126 {
		return true
	}
	return false
}

func tryHandleKeyPress(ev termbox.Event) bool {
	defer render()
	if !valid(ev) {
		return true
	}
	switch mode {
	case ModeNormal:
		if ev.Ch == ':' {
			mode = ModeCommand
			commandBuffer = []rune{}
		} else if ev.Ch == 'i' {
			mode = ModeInsert
		} else if ev.Ch == 'a' {
			mode = ModeInsert
			if lineLength(cursorY+viewOffsetY) > 0 {
				cursorX++
			}
		} else if ev.Ch == 'h' {
			moveCursorLeft()
		} else if ev.Ch == 'j' {
			moveCursorDown(false)
		} else if ev.Ch == 'k' {
			moveCursorUp()
		} else if ev.Ch == 'l' {
			moveCursorRight()
		}
		return true
	case ModeInsert:
		if ev.Key == termbox.KeyEsc {
			mode = ModeNormal
			moveCursorLeft()
			return true
		}
		return false
	case ModeCommand:
		if ev.Key == termbox.KeyEnter {
			command := string(commandBuffer)
			if command == "q" {
				termbox.Close()
				exit()
			} else if command == "w" {
				saveBufferToFile()
			} else if command == "x" || command == "wq" {
				saveBufferToFile()
				termbox.Close()
				exit()
			}
			mode = ModeNormal
			return true
		}
		if ev.Key == termbox.KeyEsc {
			mode = ModeNormal
			return true
		}
		if ev.Key == termbox.KeyBackspace || ev.Key == termbox.KeyBackspace2 {
			if len(commandBuffer) > 0 {
				commandBuffer = commandBuffer[:len(commandBuffer)-1]
			}
			return true
		}
		if ev.Ch != 0 {
			commandBuffer = append(commandBuffer, ev.Ch)
		}
		return true
	}
	return false
}

func handleKeyPress(ev termbox.Event) {
	defer render()
	ctx := context.Background()
	switch mode {
	case ModeInsert:
		var newBuffer []rune
		if ev.Key == termbox.KeyEsc {
			mode = ModeNormal
			moveCursorLeft()
			return
		}
		if ev.Key == termbox.KeyEnter {
			insertIndex := lineStarts[cursorY+viewOffsetY] + cursorX + viewOffsetX
			if insertIndex > len(buffer) {
				insertIndex = len(buffer)
			}
			newBuffer = append([]rune(nil), buffer[:insertIndex]...)
			newBuffer = append(newBuffer, '\n')
			newBuffer = append(newBuffer, buffer[insertIndex:]...)
			updateLineStarts()
			moveCursorDown(true)
		} else if ev.Key == termbox.KeyBackspace || ev.Key == termbox.KeyBackspace2 {
			if len(buffer) > 0 {
				if cursorX == 0 && cursorY > 0 {
					newlineIndexToRemove := lineStarts[cursorY+viewOffsetY] - 1
					if newlineIndexToRemove >= 0 && newlineIndexToRemove < len(buffer) {
						// Remove the newline character from the buffer
						newBuffer = append(buffer[:newlineIndexToRemove], buffer[newlineIndexToRemove+1:]...)
						updateLineStarts()

						// Update cursor position
						cursorY--
						cursorX = newlineIndexToRemove - lineStarts[cursorY+viewOffsetY]
					}
				} else if cursorX > 0 {
					// Calculate the insertion index
					insertIndex := lineStarts[cursorY+viewOffsetY] + cursorX + viewOffsetX
					if insertIndex > len(buffer) {
						insertIndex = len(buffer)
					}
					newBuffer = append([]rune(nil), buffer[:insertIndex-1]...)
					newBuffer = append(newBuffer, buffer[insertIndex:]...)
					updateLineStarts()
					moveCursorLeft()
				}
			}
		} else if ev.Key == termbox.KeySpace {
			// Calculate the insertion index
			insertIndex := 0
			if len(lineStarts) > 0 {
				insertIndex = lineStarts[cursorY+viewOffsetY] + cursorX + viewOffsetX
			}
			if insertIndex > len(buffer) {
				insertIndex = len(buffer)
			}
			newBuffer = append([]rune(nil), buffer[:insertIndex]...)
			newBuffer = append(newBuffer, ' ')
			newBuffer = append(newBuffer, buffer[insertIndex:]...)
			updateLineStarts()
			cursorX++
			w, _ := termbox.Size()
			if cursorX > w-1 {
				viewOffsetX++
				cursorX = w - 1
			}
		} else if ev.Ch != 0 {
			// Calculate the insertion index
			insertIndex := 0
			if len(lineStarts) > 0 {
				insertIndex = lineStarts[cursorY+viewOffsetY] + cursorX + viewOffsetX
			}
			if insertIndex > len(buffer) {
				insertIndex = len(buffer)
			}
			newBuffer = append([]rune(nil), buffer[:insertIndex]...)
			newBuffer = append(newBuffer, ev.Ch)
			newBuffer = append(newBuffer, buffer[insertIndex:]...)
			updateLineStarts()
			if cursorX < 0 {
				cursorX = 0
			}
			cursorX++
			w, _ := termbox.Size()
			if cursorX > w-1 {
				viewOffsetX++
				cursorX = w - 1
			}
		}
		diff := diff(buffer, newBuffer)

		client.Do(ctx, diff)
	}
	render()
}

func saveBufferToFile() {
	content := string(buffer)
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		log.Printf("Error writing to file: %s", err)
	}
}

func moveCursorUp() {
	if cursorY == 0 {
		return
	}
	cursorY--
	length := lineLength(cursorY + viewOffsetY)
	if cursorX+viewOffsetX >= length && length > viewOffsetX {
		cursorX = length - 1 - viewOffsetX
	} else if length <= viewOffsetX {
		viewOffsetX = length - 1
		if viewOffsetX < 0 {
			viewOffsetX = 0
		}
		cursorX = 0
	}
	if viewOffsetY > 0 && cursorY < viewOffsetY {
		viewOffsetY--
	}
}

func moveCursorDown(startFromBegin bool) {
	if cursorY+viewOffsetY >= len(lineStarts)-1 {
		return
	}
	if startFromBegin {
		cursorX = 0
		viewOffsetX = 0
	}
	cursorY++
	_, h := termbox.Size()
	if cursorY > h-2 {
		viewOffsetY++
		cursorY = h - 2
	}
	length := lineLength(cursorY + viewOffsetY)
	if cursorX+viewOffsetX >= length && length > viewOffsetX {
		cursorX = length - 1 - viewOffsetX
	} else if length <= viewOffsetX {
		viewOffsetX = length - 1
		cursorX = 0
	}
	if viewOffsetX < 0 {
		viewOffsetX = 0
	}
}

func moveCursorLeft() {
	if cursorX > 0 {
		cursorX--
		return
	}
	if viewOffsetX > 0 {
		viewOffsetX--
	}
}

func moveCursorRight() {
	if cursorX+viewOffsetX+1 < lineLength(cursorY+viewOffsetY) {
		width, _ := termbox.Size()
		if cursorX+1 < width {
			cursorX++
			return
		}
		viewOffsetX++
	}
}

func lineLength(line int) int {
	if line > len(lineStarts)-2 {
		return 0
	}
	// Situation where the buffer is empty
	if len(buffer) == 0 {
		return 0
	}
	// Check whether last character of the line is a newline
	if buffer[lineStarts[line+1]-1] == '\n' {
		return lineStarts[line+1] - lineStarts[line] - 1
	}
	return lineStarts[line+1] - lineStarts[line]
}

func render() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	// Display buffer
	// Track positions
	x, y := 0, 0
	w, h := termbox.Size() // Get terminal dimensions
	h--                    // Subtract 1 for the status line
	for i := 0; i < len(buffer); i++ {
		if buffer[i] == '\n' {
			y++
			x = 0
		} else {
			if y >= viewOffsetY && y < h+viewOffsetY && x >= viewOffsetX && x < w+viewOffsetX {
				termbox.SetCell(x-viewOffsetX, y-viewOffsetY, buffer[i], termbox.ColorDefault, termbox.ColorDefault)
			}
			x++
		}
		if y >= h+viewOffsetY {
			break // Stop rendering if we run out of space
		}
	}

	// If in command mode, display the command buffer and ':' prompt
	if mode == ModeCommand {
		prompt := ":"
		for i, ch := range prompt {
			termbox.SetCell(i, h, ch, termbox.ColorDefault, termbox.ColorDefault) // Display at the bottom
		}
		for i, ch := range commandBuffer {
			termbox.SetCell(i+len(prompt), h, ch, termbox.ColorDefault, termbox.ColorDefault) // Display command after the prompt
		}
	} else if mode == ModeInsert {
		// Display "-- INSERT --" at the bottom
		insert := "-- INSERT --"
		for i, ch := range insert {
			termbox.SetCell(i, h, ch, termbox.ColorDefault, termbox.ColorDefault)
		}
	}
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
	os.Exit(0)
	termbox.Close()
}

var server *Server
var client *Client

func StartEditor(path string, raftPeers string, nodes string, me int32, logPath string) {
	lineStarts = []int{0, 0}
	msgCh := make(chan interface{}, 1024)
	persister := storage.MakeGoBPersister(logPath + "/" + strconv.Itoa(int(me)))
	server = StartServer(raftpb.ParseClientEnd(raftPeers), ParseClientEnd(nodes), me, persister, msgCh)
	client = MakeClient(ParseClientEnd(nodes), me, server.ops[me])
	filePath = path
	err := termbox.Init()
	if err != nil {
		log.Fatal(err)
	}
	render()
	defer termbox.Close()

	eventQueue := make(chan termbox.Event)
	go func() {
		for {
			if event := termbox.PollEvent(); !tryHandleKeyPress(event) {
				eventQueue <- event
			}
		}
	}()

	for running {
		select {
		case ev := <-eventQueue:
			handleKeyPress(ev)
		case <-msgCh:
			buffer = server.data
			updateLineStarts()
			render()
		}
	}
}

func Clean(dataDir string) {
	os.RemoveAll(dataDir)
}
