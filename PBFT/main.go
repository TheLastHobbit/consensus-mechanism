package main

import (
	"log"
	"os"
)

const nodeCount = 4

//客户端的监听地址
var clientAddr = "127.0.0.1:8888"

//节点池，主要用来存储监听地址
var nodeTable map[string]string

/*
pbft公式： n>=3f + 1 其中n为全网总节点数量，f为最多允许的作恶、故障节点

数据从客户端输入，到接收到节点们的回复共分为5步

客户端向主节点发送请求信息
主节点N0接收到客户端请求后将请求数据里的主要信息提出，并向其余节点进行preprepare发送
从节点们接收到来自主节点的preprepare，首先利用主节点的公钥进行签名认证，其次将消息进行散列（消息摘要，以便缩小信息在网络中的传输大小）后，向其他节点广播prepare
节点接收到2f个prepare信息（包含自己）,并全部签名验证通过，则可以进行到commit步骤，向全网其他节点广播commit
节点接收到2f+1个commit信息（包含自己），并全部签名验证通过，则可以把消息存入到本地，并向客户端返回reply消息

*/

func main() {
	//为四个节点生成公私钥
	genRsaKeys()
	nodeTable = map[string]string{
		"N0": "127.0.0.1:8000",
		"N1": "127.0.0.1:8001",
		"N2": "127.0.0.1:8002",
		"N3": "127.0.0.1:8003",
	}
	if len(os.Args) != 2 {
		log.Panic("输入的参数有误！")
	}
	nodeID := os.Args[1]
	if nodeID == "client" {
		clientSendMessageAndListen() //启动客户端程序
	} else if addr, ok := nodeTable[nodeID]; ok {
		p := NewPBFT(nodeID, addr)
		go p.tcpListen() //启动节点
	} else {
		log.Fatal("无此节点编号！")
	}
	select {}
}