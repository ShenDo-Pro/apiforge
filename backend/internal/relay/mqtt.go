package relay

import (
	"net"
	"time"
)

// MQTTHandler 通过后端建立一条到 MQTT Broker 的 TCP 长连接（默认 1883），
// 将浏览器发来的 MQTT 报文（经 /ws/relay 的 WS 二进制帧）原样透传到 Broker，
// 并把 Broker 的 TCP 回包以 WS 二进制帧回传。这样前端无需直连 Broker 的 WS 端口，
// 即可借助后端 TCP 能力访问仅暴露 TCP 1883 的 Broker。
//
// 注意：为兼容 mqtt.js 浏览器端（只把 WS 二进制帧解析为 MQTT 报文），本处理器
// 成功建连后不发送任何文本控制帧，仅在连接失败时回传一条 error 文本帧，
// 其余情况靠关闭 WS 让上层（mqtt.js）感知连接断开。
type MQTTHandler struct{}

func (MQTTHandler) Name() string { return "mqtt" }

func (MQTTHandler) Serve(c Conn, target Target) {
	addr := net.JoinHostPort(target.Host, target.Port)
	remote, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		writeCtrl(c, CtrlError, err.Error())
		return
	}
	defer remote.Close()
	defer c.Close()

	// silent=true：MQTT 报文以 WS 二进制帧透传，远端关闭时不发送文本控制帧，
	// 仅关闭 WS 交由上层（mqtt.js）感知断开，避免干扰 MQTT 报文解析（M23 合并后）。
	relayStream(c, remote, true)
}
