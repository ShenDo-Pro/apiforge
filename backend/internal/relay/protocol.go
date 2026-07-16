package relay

import (
	"encoding/json"
	"io"
	"net"
	"sync"

	"github.com/gorilla/websocket"
)

// 后端 -> 前端的文本控制帧类型
const (
	CtrlConnected = "connected" // 已建立到远端的连接
	CtrlStatus    = "status"    // 带回环/远端地址信息
	CtrlError     = "error"     // 错误信息
	CtrlClosed    = "closed"    // 远端关闭连接
)

// ctrlMessage 是后端发给前端的 JSON 控制帧。
// 二进制数据帧不经过它，直接以 BinaryMessage 透传。
type ctrlMessage struct {
	Type    string `json:"type"`
	Local   string `json:"local,omitempty"`
	Remote  string `json:"remote,omitempty"`
	Message string `json:"message,omitempty"`
}

func writeCtrl(c Conn, typ, msg string) {
	b, _ := json.Marshal(ctrlMessage{Type: typ, Message: msg})
	_ = c.WriteMessage(websocket.TextMessage, b)
}

func writeStatus(c Conn, local, remote string) {
	b, _ := json.Marshal(ctrlMessage{Type: CtrlStatus, Local: local, Remote: remote})
	_ = c.WriteMessage(websocket.TextMessage, b)
}

// relayStream 在一个已建立的后端 socket 与 WS 之间做双向透传（M23：合并原 relayStream / relayStreamSilent）。
// 前端 -> 远端：WS 二进制帧直接写入 socket；文本控制帧（如心跳/关闭）忽略。
// 远端 -> 前端：socket 读取到的字节以 WS 二进制帧推送。
// silent=true 时远端关闭/出错不发送文本控制帧（MQTT 场景，避免干扰报文解析），否则发送 error/closed 控制帧。
func relayStream(c Conn, remote net.Conn, silent bool) {
	var wg sync.WaitGroup
	wg.Add(2)

	// 前端 -> 远端
	go func() {
		defer wg.Done()
		for {
			mt, data, err := c.ReadMessage()
			if err != nil {
				// 前端断开：关闭远端以中断对向读取循环
				remote.Close()
				return
			}
			if mt == websocket.TextMessage {
				// 文本帧为控制帧，当前忽略（可扩展为前端主动关闭指令）
				continue
			}
			if _, err := remote.Write(data); err != nil {
				return
			}
		}
	}()

	// 远端 -> 前端
	go func() {
		defer wg.Done()
		buf := make([]byte, 64*1024)
		for {
			n, err := remote.Read(buf)
			if n > 0 {
				if werr := c.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				if !silent {
					if err != io.EOF {
						writeCtrl(c, CtrlError, err.Error())
					} else {
						writeCtrl(c, CtrlClosed, "")
					}
				}
				return
			}
		}
	}()

	wg.Wait()
}
