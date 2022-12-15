package ws

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sunweiwe/kuber/pkg/log"
)

var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WsMessage struct {
	MessageType int
	Data        []byte
}

type WsConnection struct {
	conn    *websocket.Conn
	inChan  chan *WsMessage
	outChan chan *WsMessage
	cancel  context.CancelFunc
	lock    sync.RWMutex
	stopped bool
	OnClose func()
}

func InitWebsocket(resp http.ResponseWriter, req *http.Request) (wsConn *WsConnection, err error) {
	var conn *websocket.Conn
	if conn, err = Upgrader.Upgrade(resp, req, nil); err != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	wsConn = &WsConnection{
		conn:    conn,
		cancel:  cancel,
		lock:    sync.RWMutex{},
		inChan:  make(chan *WsMessage, 1000),
		outChan: make(chan *WsMessage, 1000),
		stopped: false,
	}
	go wsConn.wsReadLoop(ctx)
	go wsConn.wsWriteLoop(ctx)
	return
}

func (wsConn *WsConnection) wsWriteLoop(ctx context.Context) {
	var (
		message *WsMessage
		err     error
	)

	for {
		select {
		case message = <-wsConn.outChan:
			if message != nil {
				if err = wsConn.conn.WriteMessage(message.MessageType, message.Data); err != nil {
					log.Errorf("failed to write websocket message %v", err)
					wsConn.WsClose()
				}
			}
		case <-ctx.Done():
			log.Infof("stop write loop")
			return
		}
	}
}

func (wsConn *WsConnection) WsClose() {
	if wsConn.OnClose != nil {
		wsConn.OnClose()
	}

	wsConn.lock.Lock()
	defer wsConn.lock.Unlock()

	if wsConn.stopped {
		return
	}
	wsConn.stopped = true
	wsConn.cancel()
	wsConn.conn.Close()
	close(wsConn.inChan)
	close(wsConn.outChan)
}

func (wsConn *WsConnection) wsReadLoop(ctx context.Context) {
	var (
		messageType int
		data        []byte
		err         error
	)

	for {
		if messageType, data, err = wsConn.conn.ReadMessage(); err != nil {
			log.Errorf("failed to read websocket msg %v", err)
			wsConn.WsClose()
			return
		}

		xMsg := xtermMessage{}
		e := json.Unmarshal(data, &xMsg)
		if e == nil {
			if xMsg.MessageType == "close" {
				closeMsg, _ := json.Marshal(xtermMessage{
					MessageType: "input",
					Input:       "exit\r",
				})
				wsConn.inChan <- &WsMessage{MessageType: messageType, Data: closeMsg}
			}
		}
		wsConn.inChan <- &WsMessage{MessageType: messageType, Data: data}

		select {
		case <-ctx.Done():
			return
		default:
			continue
		}
	}
}

func (wsCoon *WsConnection) WsWrite(messageType int, data []byte) (err error) {
	wsCoon.lock.Lock()
	defer wsCoon.lock.Unlock()
	if wsCoon.stopped {
		err = errors.New("can't write on closed channel")
		return
	}
	wsCoon.outChan <- &WsMessage{messageType, data}
	return
}
