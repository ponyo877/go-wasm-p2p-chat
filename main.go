//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"syscall/js"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/ponyo877/go-wasm-p2p-chat/go-ayame"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

var (
	wsScheme          string
	matchmakingOrigin string
	signalingOrigin   string
)

type mmReqMsg struct {
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type mmResMsg struct {
	Type      string    `json:"type"`
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	mmURL := url.URL{Scheme: wsScheme, Host: matchmakingOrigin, Path: "/matchmaking"}
	signalingURL := url.URL{Scheme: wsScheme, Host: signalingOrigin, Path: "/signaling"}

	now := time.Now()
	userID, _ := shortHash(now)
	reqMsg, err := json.Marshal(mmReqMsg{
		UserID:    userID,
		CreatedAt: now,
	})
	if err != nil {
		log.Fatal(err)
	}
	var resMsg mmResMsg
	var dc *webrtc.DataChannel
	defer func() {
		if dc != nil {
			dc.Close()
		}
	}()
	var conn *ayame.Connection
	connected := make(chan bool)
	js.Global().Set("startNewChat", js.FuncOf(func(_ js.Value, _ []js.Value) interface{} {
		go func() {
			ws, _, err := websocket.Dial(context.Background(), mmURL.String(), nil)
			if err != nil {
				log.Fatal(err)
			}
			defer ws.Close(websocket.StatusNormalClosure, "close connection")

			if err := ws.Write(context.Background(), websocket.MessageText, reqMsg); err != nil {
				log.Fatal(err)
			}
			logElem("[Sys]: Waiting match...\n")
			for {
				if err := wsjson.Read(context.Background(), ws, &resMsg); err != nil {
					log.Fatal(err)
					break
				}
				if resMsg.Type == "MATCH" {
					break
				}
			}
			ws.Close(websocket.StatusNormalClosure, "close connection")
			if resMsg.Type == "MATCH" {
				conn = ayame.NewConnection(signalingURL.String(), resMsg.RoomID, ayame.DefaultOptions(), false, false)
				conn.OnOpen(func(metadata *interface{}) {
					log.Println("Open")
					var err error
					dc, err = conn.CreateDataChannel("matchmaking-example", nil)
					if err != nil && err != fmt.Errorf("client does not exist") {
						log.Printf("CreateDataChannel error: %v", err)
						return
					}
					log.Printf("CreateDataChannel: label=%s", dc.Label())
					dc.OnMessage(onMessage())
				})

				conn.OnConnect(func() {
					logElem("[Sys]: Matching! Start P2P chat not via server\n")
					conn.CloseWebSocketConnection()
					connected <- true
				})

				conn.OnDataChannel(func(c *webrtc.DataChannel) {
					log.Printf("OnDataChannel: label=%s", c.Label())
					if dc == nil {
						dc = c
					}
					dc.OnMessage(onMessage())
				})

				if err := conn.Connect(); err != nil {
					log.Fatal("Failed to connect Ayame", err)
				}
				select {
				case <-connected:
					return
				}
			}
		}()
		return js.Undefined()
	}))
	js.Global().Set("sendMessage", js.FuncOf(func(_ js.Value, _ []js.Value) interface{} {
		go func() {
			el := getElementByID("message")
			message := el.Get("value").String()
			if message == "" {
				js.Global().Call("alert", "Message must not be empty")
				return
			}
			if dc == nil {
				return
			}
			if err := dc.SendText(message); err != nil {
				handleError()
				return
			}
			logElem(fmt.Sprintf("[You]: %s\n", message))
			el.Set("value", "")
		}()
		return js.Undefined()
	}))
	select {}
}

func shortHash(now time.Time) (string, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(now.String())); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:7], nil
}

func onMessage() func(webrtc.DataChannelMessage) {
	return func(msg webrtc.DataChannelMessage) {
		if msg.IsString {
			logElem(fmt.Sprintf("[Any]: %s\n", msg.Data))
		}
	}
}

func logElem(msg string) {
	el := getElementByID("logs")
	el.Set("innerHTML", el.Get("innerHTML").String()+msg)
}

func handleError() {
	logElem("[Sys]: Maybe Any left, Please restart\n")
}

func getElementByID(id string) js.Value {
	return js.Global().Get("document").Call("getElementById", id)
}
