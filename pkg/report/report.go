package report

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"sync"
)

var (
	//ErrBlockNotOpen indicates close called before open
	ErrBlockNotOpen = errors.New("close called on unopened block")

	//ErrTitleInBlock indicates title write attempted in code block
	ErrTitleInBlock = errors.New("attempted title write in code block")
)

// Writer standardized  report building in markdown format
type Writer struct {
	writer      *bufio.Writer
	err         error
	blockOpened bool
	lock        sync.Mutex
}

// NewWriter create Writer with initialized buffer and arbitrary number of
// writer targets. Initializes with 1MBi buffer
func NewWriter(w ...io.Writer) *Writer {
	mw := io.MultiWriter(w...)
	return &Writer{
		writer: bufio.NewWriterSize(mw, 1024*1024),
	}
}

// Err return first error to occur when using Writer
func (rw *Writer) Err() error {
	return rw.err
}

// OpenBlock opens up a code block for writing
// CloseBlock() must be called to end the block
func (rw *Writer) OpenBlock() int {
	rw.lock.Lock()
	defer rw.lock.Unlock()
	var n int
	if rw.err != nil {
		return n
	}
	rw.blockOpened = true
	n, rw.err = rw.writer.WriteString("```\n")
	return n
}

//CloseBlock close code block
func (rw *Writer) CloseBlock() int {
	rw.lock.Lock()
	defer rw.lock.Unlock()
	var n int
	if rw.err != nil {
		return n
	}

	if !rw.blockOpened {
		rw.err = ErrBlockNotOpen
		return n
	}
	rw.blockOpened = false
	n, rw.err = rw.writer.WriteString("\n```\n")
	return n
}

// AddTitle write level 2 title
func (rw *Writer) AddTitle(msg string) int {
	rw.lock.Lock()
	defer rw.lock.Unlock()

	var n int
	if rw.err != nil {
		return n
	}

	if rw.blockOpened {
		rw.err = ErrTitleInBlock
		return n
	}

	n, rw.err = rw.writer.WriteString(fmt.Sprintf("\n## %s\n", msg))
	return n
}

func (rw *Writer) Write(msg string) int {
	rw.lock.Lock()
	defer rw.lock.Unlock()

	var n int
	if rw.err != nil {
		return n
	}

	n, rw.err = rw.writer.WriteString(fmt.Sprintf("%s\n", msg))
	return n
}

//Flush close out report
func (rw *Writer) Flush() int {
	rw.lock.Lock()
	defer rw.lock.Unlock()

	var n int
	if rw.err != nil {
		return n
	}

	rw.err = rw.writer.Flush()

	return n
}
