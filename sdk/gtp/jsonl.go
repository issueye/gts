package gtp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

type Encoder struct {
	w *bufio.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: bufio.NewWriter(w)}
}

func (e *Encoder) Encode(frame Frame) error {
	data, err := EncodeFrame(frame)
	if err != nil {
		return err
	}
	if bytes.IndexByte(data, '\n') >= 0 {
		return fmt.Errorf("gtp frame contains newline")
	}
	if _, err := e.w.Write(data); err != nil {
		return err
	}
	if err := e.w.WriteByte('\n'); err != nil {
		return err
	}
	return e.w.Flush()
}

type Decoder struct {
	r *bufio.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: bufio.NewReader(r)}
}

func (d *Decoder) Decode() (Frame, error) {
	line, err := d.r.ReadSlice('\n')
	if err != nil {
		return Frame{}, err
	}
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	return DecodeFrame(line)
}

func EncodeJSONL(frame Frame) ([]byte, error) {
	data, err := EncodeFrame(frame)
	if err != nil {
		return nil, err
	}
	if bytes.IndexByte(data, '\n') >= 0 {
		return nil, fmt.Errorf("gtp frame contains newline")
	}
	out := make([]byte, 0, len(data)+1)
	out = append(out, data...)
	out = append(out, '\n')
	return out, nil
}
