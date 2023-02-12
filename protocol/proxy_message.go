package protocol

// ProxyMessage 消息体 TODO 封装
type ProxyMessage struct {
	Type     NatxMessageType
	MetaData map[string]interface{}
	Data     []byte
}
