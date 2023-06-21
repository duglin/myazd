// From https://raw.githubusercontent.com/crdx/mission/v0.4.0/jsonc/jsonc.go

package main

import "errors"

type JsonCComment struct {
	state     State
	multiLine bool
	isJson    bool
}

type State int

const (
	stopped State = iota
	canStart
	started
	canStop
)

const (
	tab            = 9   // (\t)
	lineFeed       = 10  // (\n)
	carriageReturn = 13  // (\r)
	space          = 32  // ( )
	quote          = 34  // (")
	asterisk       = 42  // (*)
	forwardSlash   = 47  // (/)
	backSlash      = 92  // (\)
	charN          = 110 // (n)
)

func JsonCDecode(bytesIn []byte) ([]byte, error) {
	n, err := decode(bytesIn)
	if err != nil {
		return nil, err
	}

	return bytesIn[:n], nil
}

func (self *JsonCComment) reset() {
	self.state = stopped
	self.multiLine = false
}

func (self JsonCComment) isComplete() bool {
	return self.state == stopped
}

func internalDecode(bytes []byte, comment *JsonCComment) int {
	i := 0

	for _, current := range bytes {
		switch comment.state {
		case stopped:
			if current == quote {
				comment.isJson = !comment.isJson
			}

			if comment.isJson {
				bytes[i] = current
				i++
				continue
			}

			if current == space || current == tab || current == lineFeed || current == carriageReturn {
				continue
			}

			if current == forwardSlash {
				comment.state = canStart
				continue
			}

			bytes[i] = current
			i++

		case canStart:
			if current == asterisk || current == forwardSlash {
				comment.state = started
			}

			comment.multiLine = (current == asterisk)

		case started:
			if current == asterisk || current == backSlash {
				comment.state = canStop
			}

			if current == lineFeed && !comment.multiLine {
				comment.reset()
			}

		case canStop:
			if current == forwardSlash || current == charN {
				comment.reset()
			}
		}
	}

	return i
}

func decode(bytes []byte) (int, error) {
	comment := JsonCComment{}
	n := internalDecode(bytes, &comment)

	if !comment.isComplete() {
		return 0, errors.New("unexpected end of comment")
	}

	return n, nil
}
