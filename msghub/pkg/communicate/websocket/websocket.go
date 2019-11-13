package websocket

import (
	"k8s.io/klog"
	"github.com/jwzl/wssocket/server"
)

type WSServer struct {
	conf 	  *config.WebsocketServerConfig
	wsserver  *server.Server
}


func NewWSServer(conf *config.WebsocketServerConfig) *WSServer {
	if conf == nil {
		return nil
	}

	//Addr: conf.URL,
		//AutoRoute: false,
		//HandshakeTimeout: time.Duration(conf.HandshakeTimeout) * time.Second,
		//ReadDeadline:  time.Duration(conf.ReadDeadline) * time.Second,
		//WriteDeadline: time.Duration(conf.WriteDeadline) * time.Second,
	wss := &server.Server{
		Addr: conf.URL,
		AutoRoute: false,
		HandshakeTimeout: time.Duration(conf.HandshakeTimeout) * time.Second,
	}

	srv := &WSServer{
		conf: conf,
		wsserver: wss, 
	}

	
}
