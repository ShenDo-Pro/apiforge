package relay

import (
	"net"
	"time"
)

// UDPHandler 通过后端建立一条"已连接"的 UDP socket，与前端 WS 双向透传数据报。
// UDP 无连接语义，这里用 net.Dial("udp") 将 socket 绑定到指定远端，
// 后续进出均限定于该远端地址，便于在调试工具里收发数据报。
type UDPHandler struct{}

func (UDPHandler) Name() string { return "udp" }

func (UDPHandler) Serve(c Conn, target Target) {
	addr := net.JoinHostPort(target.Host, target.Port)
	remote, err := net.DialTimeout("udp", addr, 10*time.Second)
	if err != nil {
		writeCtrl(c, CtrlError, err.Error())
		return
	}
	defer remote.Close()
	defer c.Close()

	writeCtrl(c, CtrlConnected, "")
	writeStatus(c, remote.LocalAddr().String(), remote.RemoteAddr().String())

	relayStream(c, remote, false)
}
