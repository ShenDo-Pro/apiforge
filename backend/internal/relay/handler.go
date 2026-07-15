package relay

import (
	"encoding/json"
	"net/http"

	"apiforge/backend/pkg/response"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WSConn 适配 gorilla/websocket 连接以满足 relay.Conn 接口（值接收者，便于按值满足接口）。
type WSConn struct {
	*websocket.Conn
}

func (c WSConn) ReadMessage() (int, []byte, error)   { return c.Conn.ReadMessage() }
func (c WSConn) WriteMessage(mt int, b []byte) error { return c.Conn.WriteMessage(mt, b) }
func (c WSConn) Close() error                        { return c.Conn.Close() }

// Handler 是 /ws/relay 端点：按 ?proto=xxx 解析远端目标，升级 WS 后转交对应协议处理器做双向透传。
//   - tcp/udp/mqtt：目标由 ?host=&port= 指定（后端建立 socket）。
//   - ws/socketio：目标由 ?url= 指定（后端作为 WS 客户端连往目标），?headers= 为 JSON 自定义请求头，
//     借此突破浏览器 WebSocket 不能设置自定义请求头的限制，并能访问内网/需服务端代理的目标。
func Handler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	proto := q.Get("proto")
	if proto == "" {
		proto = "tcp"
	}

	var target Target
	switch proto {
	case "ws", "socketio":
		target.URL = q.Get("url")
		if target.URL == "" {
			response.Fail(w, http.StatusBadRequest, 400, "url is required for "+proto)
			return
		}
		if hdr := q.Get("headers"); hdr != "" {
			_ = json.Unmarshal([]byte(hdr), &target.Headers)
		}
		target.Subprotocol = q.Get("sub")
		target.Insecure = q.Get("insecure") == "1"
	default:
		target.Host = q.Get("host")
		target.Port = q.Get("port")
		if target.Host == "" || target.Port == "" {
			response.Fail(w, http.StatusBadRequest, 400, "host and port are required")
			return
		}
	}

	h, ok := Get(proto)
	if !ok {
		// 未注册协议：返回占位信息，便于前端明确当前能力边界
		response.Fail(w, http.StatusNotImplemented, 501, "protocol not implemented yet: "+proto)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	h.Serve(WSConn{conn}, target)
}
