package codec

import (
	"bytes"
	"github.com/proxy/go-natx/common"
	"github.com/proxy/go-natx/protocol"

	"github.com/go-netty/go-netty"
)

type NatxMessageEncoder struct {
	common.NatxCommonHandler
	netty.OutboundHandler
}

func NewNatxMessageEncoder() *NatxMessageEncoder {
	return &NatxMessageEncoder{}
}

func (p *NatxMessageEncoder) CodecName() string {
	return "NatxMessageEncoder"
}

func (p *NatxMessageEncoder) HandleWrite(ctx netty.OutboundContext, message netty.Message) {
	out := new(bytes.Buffer)
	switch message.(type) {
	case *protocol.ProxyMessage:
		p.Encode(ctx, message.(*protocol.ProxyMessage), out)
		ctx.HandleWrite(out)
	case []byte:
		ctx.HandleWrite(message)
	}
}
