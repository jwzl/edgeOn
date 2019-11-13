package msghub

import (
	"k8s.io/klog"
	"github.com/jwzl/beehive/pkg/core/context"
	"github.com/jwzl/edgeOn/msghub/pkg/config"
	"github.com/jwzl/edgeOn/msghub/pkg/communicate/mqtt"
)

type Controller struct {
	EdgeID		string
	context		*context.Context
	stopChan   chan struct{}

	//mqtt client
	mqtt	   *mqtt.MqttClient
}

func (hc * Controller)Start(){
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
		err = hc.mqtt.Start()
		if err != nil {
			klog.Fatalf("Start mqtt client err (%v)", err)
			return
		}
	}

	// Start websocket server.
} 
