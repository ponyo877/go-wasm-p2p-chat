// Package ayame は Ayame クライアントライブラリです。
package ayame

import (
	"github.com/pion/webrtc/v3"
)

var (
	turnHost string
	turnUser string
	turnPass string
)

// DefaultOptions は Ayame 接続オプションのデフォルト値を生成して返します。
func DefaultOptions() *ConnectionOptions {
	return &ConnectionOptions{
		ICEServers: []webrtc.ICEServer{
			// 本番環境では TURN サーバを指定した方が良さそうです
			// {
			// 	URLs:           []string{"turn:" + turnHost},
			// 	Username:       turnUser,
			// 	Credential:     turnPass,
			// 	CredentialType: webrtc.ICECredentialTypePassword,
			// },
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
		ClientID:     getULID(),
		UseTrickeICE: true,
	}
}

// NewConnection は Ayame Connection を生成して返します。
func NewConnection(signalingURL string, roomID string, options *ConnectionOptions, debug bool, isRelay bool) *Connection {
	transportPolicy := webrtc.ICETransportPolicyAll
	if isRelay {
		transportPolicy = webrtc.ICETransportPolicyRelay
	}

	if options == nil {
		options = DefaultOptions()
	}

	c := &Connection{
		SignalingURL:  signalingURL,
		RoomID:        roomID,
		Options:       options,
		Debug:         debug,
		AuthnMetadata: nil,

		authzMetadata:   nil,
		connectionState: webrtc.ICEConnectionStateNew,
		connectionID:    "",
		ws:              nil,
		pc:              nil,
		pcConfig: webrtc.Configuration{
			ICEServers:         options.ICEServers,
			ICETransportPolicy: transportPolicy,
		},
		isOffer:       false,
		isExistClient: false,

		dataChannels: map[string]*webrtc.DataChannel{},

		onOpenHandler:        func(metadata *interface{}) {},
		onConnectHandler:     func() {},
		onDisconnectHandler:  func(reason string, err error) {},
		onByeHandler:         func() {},
		onDataChannelHandler: func(dc *webrtc.DataChannel) {},
	}

	return c
}
