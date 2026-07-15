package relay

import (
	"crypto/tls"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// WSHandler 通过后端作为 WebSocket 客户端连往目标 ws(s) 服务，再与前端 WS 双向透传。
// 相比浏览器直连，后端中继可设置自定义请求头（浏览器 WS 不允许），并能访问内网/需服务端代理的目标。
type WSHandler struct{}

func (WSHandler) Name() string { return "ws" }

func (WSHandler) Serve(c Conn, target Target) {
	dialer := websocket.Dialer{}
	if target.Subprotocol != "" {
		dialer.Subprotocols = []string{target.Subprotocol}
	}
	if target.Insecure {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	header := http.Header{}
	for k, v := range target.Headers {
		header.Set(k, v)
	}
	remote, _, err := dialer.Dial(target.URL, header)
	if err != nil {
		writeCtrl(c, CtrlError, err.Error())
		return
	}
	defer remote.Close()
	defer c.Close()
	writeCtrl(c, CtrlConnected, "")
	relayWS(c, remote)
}

// SocketIOHandler 复用 WebSocket 隧道：后端作为 socket.io 的 WS 传输客户端连往目标，
// 将 socket.io 的 WS 帧原样透传，使前端 socket.io-client 可经本中继访问内网/自定义头的目标。
type SocketIOHandler struct{}

func (SocketIOHandler) Name() string { return "socketio" }

func (SocketIOHandler) Serve(c Conn, target Target) {
	WSHandler{}.Serve(c, target)
}

// relayWS 在两个 WS 连接之间双向透传，保留消息类型（文本/二进制）。
// 两个参数均为 relay.Conn 接口：前端侧为 WSConn，目标侧为 gorilla 的 *websocket.Conn，
// 二者都满足 ReadMessage/WriteMessage/Close，故可统一透传。
func relayWS(local, remote Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	// 前端 -> 目标
	go func() {
		defer wg.Done()
		for {
			mt, data, err := local.ReadMessage()
			if err != nil {
				remote.Close()
				return
			}
			if err := remote.WriteMessage(mt, data); err != nil {
				return
			}
		}
	}()

	// 目标 -> 前端
	go func() {
		defer wg.Done()
		for {
			mt, data, err := remote.ReadMessage()
			if err != nil {
				local.Close()
				return
			}
			if err := local.WriteMessage(mt, data); err != nil {
				return
			}
		}
	}()

	wg.Wait()
}
