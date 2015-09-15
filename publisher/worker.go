package publisher

import (
	"sync"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/outputs"
)

type messageWorker struct {
	queue   chan message
	ws      *workerSignal
	handler messageHandler
}

type message struct {
	signal outputs.Signaler
	event  common.MapStr
	events []common.MapStr
}

type workerSignal struct {
	done chan struct{}
	wg   sync.WaitGroup
}

type messageHandler interface {
	onMessage(m message)
}

func newMessageWorker(ws *workerSignal, hwm int, h messageHandler) *messageWorker {
	p := &messageWorker{}
	p.init(ws, hwm, h)
	return p
}

func (p *messageWorker) init(ws *workerSignal, hwm int, h messageHandler) {
	p.queue = make(chan message, hwm)
	p.ws = ws
	p.handler = h
	ws.wg.Add(1)
	go p.run()
}

func (p *messageWorker) run() {
	defer func() {
		close(p.queue)
		for msg := range p.queue { // clear queue
			outputs.SignalFailed(msg.signal)
		}
		p.ws.wg.Done()
	}()

	for {
		select {
		case m := <-p.queue:
			p.handler.onMessage(m)
		case <-p.ws.done:
			return
		}
	}
}

func (p *messageWorker) send(m message) {
	p.queue <- m
}

func (ws *workerSignal) stop() {
	close(ws.done)
	ws.wg.Wait()
}

func (ws *workerSignal) Init() {
	ws.done = make(chan struct{})
}
