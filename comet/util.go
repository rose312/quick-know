package comet

import (
	"github.com/Alienero/spp"
)

// Tcp write queue
type PackQueue struct {
	// The last error in the tcp connection
	writeError error
	// Notice read the error
	errorChan chan error

	writeChan chan *spp.Pack
	readChan  chan *packAndErr
	// Pack connection
	rw *spp.Conn
}
type packAndErr struct {
	pack *spp.Pack
	err  error
}

func NewPackQueue(rw *spp.Conn) *PackQueue {
	return &PackQueue{
		rw:        rw,
		writeChan: make(chan *spp.Pack, Conf.WirteLoopChanNum),
		readChan:  make(chan *packAndErr, 1),
		errorChan: make(chan error, 1),
	}
}
func (queue *PackQueue) writeLoop() {
	// defer recover()
	var err error
loop:
	for {
		select {
		case pack := <-queue.writeChan:
			if pack == nil {
				break loop
			}
			err = queue.rw.WritePack(pack)
			if err != nil {
				// Tell listen error
				queue.writeError = err
				break loop
			}
		}
	}
	// Notice the read
	if err != nil {
		queue.errorChan <- err
	}
}

// Server write queue
func (queue *PackQueue) WritePack(pack *spp.Pack) error {
	if queue.writeError != nil {
		return queue.writeError
	}
	queue.writeChan <- pack
	return nil
}
func (queue *PackQueue) ReadPack() (pack *spp.Pack, err error) {
	go func() {
		p := new(packAndErr)
		p.pack, p.err = queue.rw.ReadPack()
		queue.readChan <- p
	}()
	select {
	case err = <-queue.errorChan:
		// Hava an error
		// pass
	case pAndErr := <-queue.readChan:
		pack = pAndErr.pack
		err = pAndErr.err
	}
	return
}

// Only call once
func (queue *PackQueue) ReadPackInLoop(fin <-chan byte) <-chan *packAndErr {
	ch := make(chan *packAndErr, Conf.ReadPackLoop)
	go func() {
		// defer recover()
		p := new(packAndErr)
	loop:
		for {
			p.pack, p.err = queue.rw.ReadPack()
			select {
			case ch <- p:
				if p.err != nil {
					break loop
				}
			case <-fin:
				break loop
			}
			p = new(packAndErr)
		}
		close(ch)
	}()
	return ch
}
func (queue *PackQueue) Close() error {
	close(queue.writeChan)
	close(queue.readChan)
	close(queue.errorChan)
	return nil
}
