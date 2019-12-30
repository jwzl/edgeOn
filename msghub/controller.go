package msghub

import (
	"strings"
	"k8s.io/klog"
	"github.com/jwzl/wssocket/model"
	"github.com/jwzl/beehive/pkg/core/context"
	"github.com/jwzl/edgeOn/msghub/types"
	"github.com/jwzl/edgeOn/msghub/config"
	"github.com/jwzl/edgeOn/msghub/communicate/mqtt"
	"github.com/jwzl/edgeOn/msghub/communicate/websocket"
)

type Controller struct {
	EdgeID		string
	context		*context.Context
	stopChan   chan struct{}
	//mqtt client
	mqtt	   *mqtt.MqttClient
	//websocket server.
	wsServer   *websocket.WSServer	
}

func NewController(ctx *context.Context) *Controller {
	return &Controller{
		context:	ctx,
		stopChan:   make(chan struct{}),
	}	
}

func (hc * Controller)Start(){
	klog.Infof("Start the hub....")	

	conf, err := config.GetMqttConfig()
	if err != nil {
		klog.Warningf("failed to get mqtt configuration: %v", err)
	}else {
		client := mqtt.NewMqttClient(conf)
		if client == nil {
			klog.Fatalf("failed Create mqtt client, please check your conf file")
			return
		}
	
		hc.mqtt = client
		
		//Start the mqtt client.	
		go hc.mqtt.Start()
		
	}

	// Start websocket server.
	wsConf, err := config.GetWSServerConfig()
	if err != nil {
		klog.Errorf("failed to get websocket configuration: %v", err)
		return
	}
	server := websocket.NewWSServer(wsConf)
	if server == nil {
		klog.Fatalf("failed Create websocket server, please check your conf file")
		return
	}

	hc.wsServer = server
	
	//Start the websocket server.
	go hc.wsServer.Start()
		
	stop := make(chan struct{}, 3)

	go 	hc.routeToUpstream(stop)
	go 	hc.routeFromWebsocket(stop)
	go  hc.routeFromMqtt(stop)

	<-stop
	if hc.mqtt != nil {
		hc.mqtt.Close()
	}
	hc.wsServer.Close()
} 

func (hc * Controller) routeToUpstream(stop chan struct{}){
	for {
		v, err := hc.context.Receive(types.HubModuleName)
		if err != nil {
			klog.Errorf("failed to receive message from edge: %v", err)
			stop <- struct{}{}
			break
		}

		msg, isThisType := v.(*model.Message)
		if !isThisType || msg == nil {
			//invalid message type or msg == nil, Ignored. 		
			continue
		}

		target := msg.GetTarget()
		if strings.Contains(target, types.CloudName) {
			// Send message over mqtt.
			hc.mqtt.WriteMessage(msg) 
		}

		if strings.Contains(target, types.EdgeAppName) {
			msgChan := hc.wsServer.GetMessageChan(false)
			msgChan <- msg
		}
	}
}

func (hc * Controller) routeFromWebsocket(stop chan struct{}){
	for {
		msg, ok := <-hc.wsServer.GetMessageChan(true)
		if !ok {
			klog.Errorf("failed to receive message from ws channel")
			stop <- struct{}{}
			break
		}

		if msg == nil {
			//msg == nil, Ignored. 		
			continue
		}

		target := msg.GetTarget()
		if strings.Contains(target, types.TwinModuleName) {
			hc.context.Send(types.TwinModuleName, msg)
		}

		if strings.Contains(target, types.CloudName) {
			// Send message over mqtt.
			hc.mqtt.WriteMessage(msg) 
		}

		if strings.Contains(target, types.EdgeAppName) {
			msgChan := hc.wsServer.GetMessageChan(false)
			msgChan <- msg
		}
	}
}

func (hc * Controller) routeFromMqtt(stop chan struct{}){
	for {
		msg, err := hc.mqtt.ReadMessage()
		if err != nil {
			klog.Errorf("failed to receive message from mqtt channel")
			stop <- struct{}{}
			break
		}

		if msg == nil {
			//msg == nil, Ignored. 		
			continue
		}

		target := msg.GetTarget()
		if strings.Contains(target, types.TwinModuleName) {
			hc.context.Send(types.TwinModuleName, msg)
		}

		if strings.Contains(target, types.CloudName) {
			// Send message over mqtt.
			hc.mqtt.WriteMessage(msg) 
		}

		if strings.Contains(target, types.EdgeAppName) {
			msgChan := hc.wsServer.GetMessageChan(false)
			msgChan <- msg
		}
	}
}
