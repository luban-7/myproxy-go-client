package handler

import (
	"github.com/go-netty/go-netty"
	"github.com/proxy/go-natx/codec"
	"github.com/proxy/go-natx/common"
	"github.com/proxy/go-natx/net"
	"github.com/proxy/go-natx/protocol"
	"log"
	"sync"
)

// ProxyClientHandler 客户端处理器
type ProxyClientHandler struct {
	netty.ActiveHandler
	common.NatxCommonHandler
	userId            string
	channelHandlerMap *ProxyHandlerMap
}

type ProxyHandlerMap struct {
	sync.Mutex
	m map[string]*LocalProxyHandler
}

func initProxyHandlerMap() *ProxyHandlerMap {
	return &ProxyHandlerMap{
		m: make(map[string]*LocalProxyHandler),
	}
}

func (p *ProxyHandlerMap) Set(channelId string, proxyHandler *LocalProxyHandler) {
	p.Lock()
	defer p.Unlock()
	p.m[channelId] = proxyHandler
}
func (p *ProxyHandlerMap) Get(channelId string) *LocalProxyHandler {
	p.Lock()
	defer p.Unlock()
	return p.m[channelId]
}
func (p *ProxyHandlerMap) Del(channelId string) {
	p.Lock()
	defer p.Unlock()
	delete(p.m, channelId)
}

func NewProxyClientHandler(userId string) netty.InboundHandler {
	return &ProxyClientHandler{
		channelHandlerMap: initProxyHandlerMap(),
		userId:            userId,
	}
}

// HandleActive 代理客户端建立与服务端链接时调用，向代理服务端发送自己的userId，注册启动
func (p *ProxyClientHandler) HandleActive(ctx netty.ActiveContext) {

	metaData := make(map[string]interface{})
	metaData["userId"] = p.userId

	natxMessage := &protocol.ProxyMessage{
		Type:     protocol.REGISTER_TO_SERVER,
		MetaData: metaData,
	}
	ctx.Write(natxMessage)
	p.NatxCommonHandler.HandleActive(ctx)
}

// HandleEvent 向服务端发送心跳包
func (p *ProxyClientHandler) HandleEvent(ctx netty.EventContext, event netty.Event) {
	switch event.(type) {
	case netty.WriteIdleEvent:
		metaData := make(map[string]interface{})
		metaData["userId"] = p.userId
		natxMessage := &protocol.ProxyMessage{
			Type:     protocol.KEEPALIVE_TO_SERVER,
			MetaData: metaData,
		}
		ctx.Write(natxMessage)
	case netty.ReadIdleEvent:
		ctx.Close(nil)
	}
}

// HandleRead 接收到代理服务端数据包时调用，根据数据包类型（请求type）分发不同方法处理
func (p *ProxyClientHandler) HandleRead(ctx netty.InboundContext, message netty.Message) {
	natxMessage := message.(*protocol.ProxyMessage)
	if natxMessage.Type == protocol.REGISTER_RESULT_TO_CLIENT {
		p.processRegisterResult(natxMessage)
	} else if natxMessage.Type == protocol.REAL_CONNECTED_TO_CLIENT {
		p.processConnected(natxMessage)
	} else if natxMessage.Type == protocol.REAL_DISCONNECTED_TO_CLIENT {
		p.processDisconnected(natxMessage)
	} else if natxMessage.Type == protocol.REAL_DATA_TO_CLIENT {
		p.processData(natxMessage)
	} else if natxMessage.Type == protocol.KEEPALIVE_TO_CLIENT {
		// 心跳包, 不处理
	} else {
	}
}

// HandleWrite 写数据
func (p *ProxyClientHandler) HandleWrite(ctx netty.OutboundContext, message netty.Message) {
	ctx.HandleWrite(message)
}

/*
 代理服务端发来注册结果消息
*/
func (p *ProxyClientHandler) processRegisterResult(proxyMessage *protocol.ProxyMessage) {
	messageType := proxyMessage.Type
	if messageType == 5 {
		log.Printf("注册成功")
	} else {
		log.Printf("注册失败，原因: %v \n", proxyMessage.MetaData["reason"])
		p.Ctx.Close(nil)
	}
}

/*
 代理服务端发来建立代理链路消息，代理客户端建立向被代理服务的链接
*/
func (p *ProxyClientHandler) processConnected(natxMessage *protocol.ProxyMessage) {
	ch := make(chan struct{})
	handler := p

	// message 携带的消息
	channelId := natxMessage.MetaData["channelId"].(string)
	serverId := natxMessage.MetaData["serverId"].(string)
	targetHost := natxMessage.MetaData["targetHost"].(string)
	targetPort := natxMessage.MetaData["targetPort"].(float64)
	//log.Printf("channelId：%v，serverId：%v，targetHost：%v，targetPort：%v \n", channelId, serverId, targetHost, targetPort)

	go func() {
		localConnection := net.NewTcpConnection()
		localConnection.Connect(targetHost, int(targetPort), func(channel netty.Channel) {
			localProxyHandler := NewLocalProxyHandler(p, natxMessage.MetaData["channelId"].(string), serverId)
			channel.Pipeline().
				//AddLast(frame.LengthFieldCodec(binary.BigEndian, 1048576, 0, 4, 0, 4)).
				AddLast(codec.NewByteArrayDecoder()).
				AddLast(codec.NewByteArrayEncoder()).
				AddLast(localProxyHandler)
			handler.channelHandlerMap.Set(channelId, localProxyHandler)
			log.Printf("LocalProxyHandler Connect , %v \n", channelId)
		}, ch)
		// TODO 失败没有去除Map中的内容？
		//natxMessage := &protocol.ProxyMessage{
		//	Type:     protocol.REAL_DISCONNECTED_TO_SERVER,
		//	MetaData: map[string]interface{}{"channelId": channelId},
		//}
		//handler.Ctx.Write(natxMessage)
	}()
	<-ch
}

/*
 代理服务端发来停止代理链路消息，代理客户端需要停止链接向被代理服务的链接
*/
func (p *ProxyClientHandler) processDisconnected(natxMessage *protocol.ProxyMessage) {
	channelId := natxMessage.MetaData["channelId"].(string)
	defer p.channelHandlerMap.Del(channelId)
	handler := p.channelHandlerMap.Get(channelId)
	//log.Printf("LocalProxyHandler disconnected   %v \n", channelId)
	if handler != nil {
		handler.Ctx.Close(nil)
	}
}

/*
 代理服务端发来访问者数据消息，代理客户端需要转发给被代理服务
*/
func (p *ProxyClientHandler) processData(proxyMessage *protocol.ProxyMessage) {
	channelId := proxyMessage.MetaData["channelId"].(string)
	handler := p.channelHandlerMap.Get(channelId)
	if nil != handler {
		handler.Ctx.Write(proxyMessage.Data)
	}
}
