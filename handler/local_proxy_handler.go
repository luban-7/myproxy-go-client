package handler

import (
	"github.com/go-netty/go-netty"
	"github.com/proxy/go-natx/common"
	"github.com/proxy/go-natx/protocol"
)

type LocalProxyHandler struct {
	common.NatxCommonHandler
	proxyHandler    *ProxyClientHandler
	remoteChannelId string
	serverId        string
}

func NewLocalProxyHandler(proxyHandler *ProxyClientHandler, remoteChannelId string, serverId string) *LocalProxyHandler {
	return &LocalProxyHandler{
		proxyHandler:    proxyHandler,
		remoteChannelId: remoteChannelId,
		serverId:        serverId,
	}
}
func (p *LocalProxyHandler) HandleRead(ctx netty.InboundContext, message netty.Message) {
	metaData := make(map[string]interface{})
	metaData["channelId"] = p.remoteChannelId
	metaData["serverId"] = p.serverId
	natxMessage := &protocol.ProxyMessage{
		Type:     protocol.REAL_DATA_TO_SERVER,
		Data:     message.([]byte),
		MetaData: metaData,
	}

	p.proxyHandler.Ctx.Write(natxMessage)
}

func (p *LocalProxyHandler) HandleActive(ctx netty.ActiveContext) {
	p.NatxCommonHandler.HandleActive(ctx)
}
func (p *LocalProxyHandler) HandleEvent(ctx netty.EventContext, event netty.Event) {
	ctx.HandleEvent(event)
}

// HandleInactive 被代理服务服务端断开连接：通知代理服务端访问者断开连接（服务端需要断开与访问者链接
func (p *LocalProxyHandler) HandleInactive(ctx netty.InactiveContext, ex netty.Exception) {
	metaData := make(map[string]interface{})
	metaData["channelId"] = p.remoteChannelId
	metaData["serverId"] = p.serverId
	message := &protocol.ProxyMessage{
		Type:     protocol.REAL_DISCONNECTED_TO_SERVER,
		MetaData: metaData,
	}
	p.proxyHandler.Ctx.Write(message)
}
