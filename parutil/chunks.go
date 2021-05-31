package parutil

import (
	"errors"
	"io"
)

type chunkState struct {
	reader eagerReader
	buf    []byte
	n      int
	eof    bool
}

func (cs chunkState) readIfNecessary() error {
	if cs.eof {
		return nil
	}

	var err error
	cs.n, err = cs.reader.Read(cs.buf)
	if err == io.EOF {
		cs.eof = true
	} else if err != nil {
		return err
	}
	return nil
}

func (cs chunkState) padIfNecessary(maxReadBytes int) {
	if !cs.eof {
		return
	}

	for i := cs.n; i < len(cs.buf); i++ {
		cs.buf[i] = 0
	}

	cs.n = 0
}

func processReadChunks(readers []io.Reader, bufs [][]byte, chunkProcessor func(int) error) error {
	if len(readers) != len(bufs) {
		return errors.New("different reader and buf counts")
	}
	if len(readers) == 0 {
		return errors.New("no readers")
	}
	bufByteCount := len(bufs[0])
	if bufByteCount == 0 {
		return errors.New("zero-length buffers")
	}
	for _, buf := range bufs {
		if len(buf) != bufByteCount {
			return errors.New("bufs with differing byte counts")
		}
	}

	chunkStates := make([]chunkState, len(readers))
	for i, reader := range readers {
		chunkStates[i] = chunkState{eagerReader{reader}, bufs[i], 0, false}
	}

	for {
		// Loop invariant: there's at least one chunkState not
		// in eof.
		maxReadBytes := 0
		for _, cs := range chunkStates {
			err := cs.readIfNecessary()
			if err != nil {
				return err
			}
			if cs.n > maxReadBytes {
				maxReadBytes = cs.n
			}
		}

		for _, cs := range chunkStates {
			cs.padIfNecessary(maxReadBytes)
		}

		// TODO: check for maxReadBytes == 0.
		err := chunkProcessor(maxReadBytes)
		if err != nil {
			return err
		}

		stillReading := false
		for _, cs := range chunkStates {
			if !cs.eof {
				stillReading = true
				break
			}
		}

		if !stillReading {
			break
		}
	}

	return nil
}
