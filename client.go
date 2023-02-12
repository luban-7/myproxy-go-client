package main

import (
	"encoding/binary"
	"fmt"
	"github.com/go-ini/ini"
	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/codec/frame"
	"github.com/go-netty/go-netty/transport/tcp"
	"github.com/proxy/go-natx/codec"
	"github.com/proxy/go-natx/handler"
	"log"
	"math"
	"time"
)

func Start(serverAddress string, serverPort int, userId string) {

	for {
		var channelInitializer = func(channel netty.Channel) {
			natxClientHandler := handler.NewProxyClientHandler(userId)
			pipeline := channel.Pipeline()
			pipeline.AddLast(codec.NewLogDecoder()).
				AddLast(frame.LengthFieldCodec(binary.BigEndian, math.MaxInt64, 0, 4, 0, 4)).
				AddLast(codec.NewNatxMessageDecoder()).
				AddLast(codec.NewNatxMessageEncoder()).
				AddLast(netty.ReadIdleHandler(60 * time.Second)).
				AddLast(netty.WriteIdleHandler(30 * time.Second)).
				AddLast(natxClientHandler)
		}

		tcpOptions := &tcp.Options{
			Timeout:         time.Second * 3,
			KeepAlive:       true,
			KeepAlivePeriod: time.Second * 5,
			Linger:          0,
			NoDelay:         true,
			SockBuf:         1024,
		}

		// 新版本go-netty启动修改
		bootstrap := netty.NewBootstrap(netty.WithClientInitializer(channelInitializer), netty.WithTransport(tcp.New()))
		channel, err := bootstrap.Connect(fmt.Sprintf("tcp://%v:%v", serverAddress, serverPort), "",
			tcp.WithOptions(tcpOptions))
		if nil != err {
			log.Println(err)
		}
		<-channel.Context().Done()
		log.Println("tunnel close...")
	}
}

func main() {
	cfgs, err := ini.Load("./conf.ini")
	if err != nil {
		fmt.Println(err)
	}
	client := cfgs.Section("client")
	userId := client.Key("userId").String()
	proxyHost := client.Key("proxyHost").String()
	proxyPort, _ := client.Key("proxyPort").Int()
	Start(proxyHost, proxyPort, userId)
}
