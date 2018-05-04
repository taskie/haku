package haku

import (
	"bytes"
	"fmt"
	"io"
	"sync"
)

// Persister persists contents of io.Reader
type Persister struct {
	Mutex  sync.Mutex
	Reader io.Reader
	Buffer bytes.Buffer
	Bytes  []byte
	Status string
}

const (
	PersisterStatusInitial = ""
	PersisterStatusReading = "reading"
	PersisterStatusRead    = "read"
)

func (p *Persister) Read(bs []byte) (int, error) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	switch p.Status {
	case PersisterStatusRead:
		return 0, io.EOF
	default:
		p.Status = PersisterStatusReading
		n, err := p.Reader.Read(bs)
		if n != 0 {
			fmt.Printf("%d %v\n", n, bs[:n])
			p.Buffer.Write(bs[:n])
		}
		if err == io.EOF {
			p.Bytes = p.Buffer.Bytes()
			p.Status = PersisterStatusRead
			p.Buffer.Reset()
		}
		return n, err
	}
}

func (p *Persister) CreatePseudoBuffer() bytes.Buffer {
	buffer := bytes.Buffer{}
	buffer.Write(p.Bytes)
	return buffer
}

func NewPersister(r io.Reader) Persister {
	return Persister{
		Reader: r,
		Status: PersisterStatusInitial,
	}
}
