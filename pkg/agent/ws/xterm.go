package ws

import (
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"

	"github.com/gorilla/websocket"
	"github.com/sunweiwe/kuber/pkg/log"
	"k8s.io/client-go/tools/remotecommand"
)

type StreamHandler struct {
	WsConn      *WsConnection
	ResizeEvent chan remotecommand.TerminalSize
}

type xtermMessage struct {
	MessageType string `json:"type"`
	Input       string `json:"input"`
	Rows        uint16 `json:"rows"`
	Cols        uint16 `json:"cols"`
}

func (handler *StreamHandler) Next() (size *remotecommand.TerminalSize) {
	ret := <-handler.ResizeEvent
	size = &ret
	return
}

func (handler *StreamHandler) Write(p []byte) (size int, err error) {
	copyData := make([]byte, len(p))
	copy(copyData, p)
	size = len(copyData)
	err = handler.WsConn.WsWrite(websocket.TextMessage, checkUTF8(copyData))
	if err != nil {
		log.Error(err, "write websocket")
		handler.WsConn.WsClose()
	}
	return
}

func (handler *StreamHandler) Read(p []byte) (size int, err error) {
	var (
		message      *WsMessage
		xtermMessage xtermMessage
	)
	if message, err = handler.WsConn.WsRead(); err != nil {
		log.Error(err, "read websocket")
		handler.WsConn.WsClose()
		return
	}

	if message == nil {
		return
	}

	if err = json.Unmarshal([]byte(message.Data), &xtermMessage); err != nil {
		log.Error(err, "unmarshal websocket message")
		return
	}

	switch xtermMessage.MessageType {
	case "resize":
		handler.ResizeEvent <- remotecommand.TerminalSize{Width: xtermMessage.Cols, Height: xtermMessage.Rows}
	case "input":
		size = len(xtermMessage.Input)
		copy(p, xtermMessage.Input)
	case "close":
		handler.WsConn.WsClose()
		err = io.EOF
	}
	return
}

func (wsConn *WsConnection) WsRead() (message *WsMessage, err error) {
	wsConn.lock.Lock()
	defer wsConn.lock.Unlock()
	if wsConn.stopped {
		err = errors.New("can't read on closed channel")
		return
	}
	message = <-wsConn.inChan
	return
}

func checkUTF8(arr []byte) []byte {
	ret := []rune{}
	for len(arr) > 0 {
		r, size := utf8.DecodeRune(arr)
		arr = arr[size:]
		ret = append(ret, r)
	}
	return []byte(string(ret))
}
