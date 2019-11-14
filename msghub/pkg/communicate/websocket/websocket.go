package websocket

import (
	"k8s.io/klog"
	"github.com/jwzl/wssocket/server"
)

type WSServer struct {
	conns         	  *sync.Map
	connLocks         sync.Map
	messageChan		  chan *model.Message	
	conf 	  		  *config.WebsocketServerConfig
	wsserver  		  *server.Server
}

// NewWSServer Create  websocket server.
func NewWSServer(ch chan *model.Message, conf *config.WebsocketServerConfig) *WSServer {
	if conf == nil || ch == nil {
		return nil
	}

	var connMap sync.Map
	srv := &WSServer{
		conns:			&connMap,
		messageChan:	ch,
		conf: 			conf,
	}
	tlsConfig, err := server.CreateTLSConfig(conf.CaFilePath, conf.CertFilePath, conf.KeyFilePath)
	if err != nil {
		klog.Errorf("Create tlsconfig err, %v", err)
		return nil
	}

	wss := &server.Server{
		Addr: conf.URL,
		AutoRoute: true,
		HandshakeTimeout: time.Duration(conf.HandshakeTimeout) * time.Second,
		TLSConfig: tlsConfig,
		ConnNotify: srv.OnConnect,
		Handler:	srv.MessageHandle,	
	}

	srv.wsserver = wss 
	
	return srv	
}

func (wss *WSServer)Start(){
	klog.Infof("Start the websocket server, listen: %s.....", wss.conf.URL)
	wss.wsserver.StartServer("", "")
} 

func (wss *WSServer) MessageHandle(Header http.Header, msg *model.Message, c *conn.Connection){
	appID := Headers.Get("app_id")
}

func (wss *WSServer) OnConnect(connection conn.Connection){
	//Record the connection.
	appID := connection.ConnectionState().Headers.Get("app_id")
	wss.conns.Store(appID, connection)
} 

//HubIOWrite: write message to connection.
func (wss *WSServer) HubIOWrite(appID string, msg *model.Message) error {
	conn, exist := wss.conns.Load(appID)
	if !exist {
		return errors.New("no this connection")
	}

	return conn.WriteMessage(msg) 
}
