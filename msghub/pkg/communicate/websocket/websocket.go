package websocket

import (
	"fmt"
	"sync"
	"time"
	"errors"
	"strings"
	"net/http"
	"k8s.io/klog"
	"github.com/jwzl/wssocket/server"
	"github.com/jwzl/wssocket/model"
	"github.com/jwzl/wssocket/conn"
	"github.com/jwzl/edgeOn/msghub/pkg/config"	
)

type WSServer struct {
	conns         	  *sync.Map
	connLocks         sync.Map
	//message from websocket connection	
	messageInChan		  chan *model.Message
	//message to websocket connection
	messageOutChan		  chan *model.Message
	keepaliveChannel  map[string]chan struct{}	
	conf 	  		  *config.WebsocketServerConfig
	wsserver  		  *server.Server
}

// NewWSServer Create  websocket server.
func NewWSServer(in, out chan *model.Message, conf *config.WebsocketServerConfig) *WSServer {
	if conf == nil || in == nil || out == nil {
		return nil
	}

	var connMap sync.Map
	srv := &WSServer{
		conns:			&connMap,
		messageInChan:	in,
		messageOutChan: out,
		keepaliveChannel: make(map[string]chan struct{}), 
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
	go wss.wsserver.StartServer("", "")

	// loop for send message.
	wss.messageOutLoop()
} 

func (wss *WSServer) MessageHandle(headers http.Header, msg *model.Message, c *conn.Connection){
	appID := headers.Get("app_id")

	if msg.GetOperation() == "keepalive" {
		//this is keepalive message make sure connection is alive.
		klog.Infof("Keepalive message received from local app: %s", appID)
		wss.keepaliveChannel[appID] <- struct{}{}
		return
	}

	//update the source.
	source := msg.GetSource()
	if source == "" {
		klog.Infof("msg.source is empty, Ignore...")
		return
    }
	source = fmt.Sprintf("%s/%s",source, appID)
	msg.Router.Source = source

	select {
	case wss.messageInChan <- msg:
	}	
}

func (wss *WSServer) OnConnect(connection conn.Connection){
	//Record the connection.
	appID := connection.ConnectionState().Headers.Get("app_id")
	wss.conns.Store(appID, connection)

	if _, ok := wss.keepaliveChannel[appID]; !ok {
		wss.keepaliveChannel[appID] = make(chan struct{}, 1)
	}

	go wss.serverConn(connection)
} 

func (wss *WSServer) serverConn(connection conn.Connection) {
	appID := connection.ConnectionState().Headers.Get("app_id")
	stop := make(chan int, 1)

	go wss.keepaliveCheckLoop(appID, stop)

	<-stop
	connection.Close()	
	wss.conns.Delete(appID)
	delete(wss.keepaliveChannel, appID)
} 

func (wss *WSServer) messageOutLoop() {
	for {
		msg, ok := <- wss.messageOutChan
		if !ok {
			klog.Errorf("messageOutChan is broken")
			return
		}

		if msg == nil {
			continue
		}

		target := msg.GetTarget()
		splitString := strings.Split(target, "/")
		if len(splitString) != 3 {
            klog.Warningf("Error msg.target format(%s),  Ignored", target)
        	continue
        }
		appID := splitString[2]
		//update the target.
		target = fmt.Sprintf("%s/%s",splitString[0], splitString[1])
		msg.Router.Target = target

		connection, exist := wss.conns.Load(appID)
		if !exist {
			klog.Warningf("No such connection for app(%s),  Ignored", appID)
        	continue
		}

		connection.WriteMessage(msg)
	}
}

func (wss *WSServer) keepaliveCheckLoop(appID string, stop chan int){
	for {
		keepaliveTimer := time.NewTimer(time.Duration(wss.conf.KeepaliveInterval) * time.Second)

		select {
		case <-keepaliveTimer.C:
			klog.Infof("timeout to recieve heartbeat from app %d", appID)
			stop <- 1
			return 
		case <-wss.keepaliveChannel[appID]:
			klog.Infof("connection is still alive from app%d", appID)
			keepaliveTimer.Stop()
		}
	}
} 
//HubIOWrite: write message to connection.
func (wss *WSServer) HubIOWrite(appID string, msg *model.Message) error {
	connection, exist := wss.conns.Load(appID)
	if !exist {
		return errors.New("no this connection")
	}

	return connection.WriteMessage(msg) 
}
