package  eventbus

import (
	"os"
	"fmt"
	"time"
	"strings"

	"k8s.io/klog"
	"github.com/jwzl/edgeOn/common"
	"github.com/jwzl/beehive/pkg/core"
	"github.com/jwzl/beehive/pkg/core/context"

	"github.com/jwzl/wssocket/model"
	"github.com/jwzl/edgeOn/eventbus/config"
	mqttBus "github.com/jwzl/edgeOn/eventbus/mqtt"
)

var (
	// TokenWaitTime to wait
	TokenWaitTime = 120 * time.Second
)

const (
	MqttModeInternal int = 0
	MqttModeBoth     int = 1
	MqttModeExternal int = 2
)

type EventBus struct {
	conf		*config.EventBusConfig
	MqttServer	*mqttBus.Server
	MqttClient	*mqttBus.Client
	context		*context.Context
}

// Register this module.
func Register(){	
	eb := &EventBus{}
	core.Register(eb)
}

//Name
func (eb *EventBus) Name() string {
	return common.BusModuleName
}

//Group
func (eb *EventBus) Group() string {
	return common.BusModuleName
}

//Start this module.
func (eb *EventBus) Start(c *context.Context) {
	klog.Infof("Start the module!")

	eb.conf = config.GetEventBusConfig()
	if eb.conf.MqttMode >= MqttModeBoth {
		eb.MqttClient = mqttBus.NewMqttClient(eb.conf.MqttServerExternal, c)
		eb.MqttClient.InitSubClient()
		eb.MqttClient.InitPubClient() 
		klog.Infof("Init Sub And Pub Client for externel mqtt broker %v successfully", eb.conf.MqttServerExternal)
	}

	if eb.conf.MqttMode <= MqttModeBoth {
		//launch an internal mqtt server only
		eb.MqttServer = mqttBus.NewMqttServer(eb.conf.MqttSessionQueueSize, 
											eb.conf.MqttServerInternal, 
											eb.conf.MqttRetain, eb.conf.MqttQOS, c)
		eb.MqttServer.InitInternalTopics()
		err := eb.MqttServer.Run()
		if err != nil {
			klog.Errorf("Launch internel mqtt broker failed, %s", err.Error())
			os.Exit(1)
		}
		klog.Infof("Launch internel mqtt broker %v successfully", eb.conf.MqttServerInternal)
	}

	eb.pubEdgeToDevice(c)
}

//Cleanup
func (eb *EventBus) Cleanup() {
	eb.context.Cleanup(eb.Name())
}

func (eb *EventBus) pubEdgeToDevice(c *context.Context){
	for {
		v, err := c.Receive(eb.Name())
		if err != nil {
			klog.Errorf("failed to receive message from edge: %v", err)
			break
		}

		klog.Infof("[EVENTBUS]  message arrived")
		msg, isThisType := v.(*model.Message)
		if !isThisType || msg == nil {
			//invalid message type or msg == nil, Ignored. 		
			continue
		}
		
		s := msg.GetSource()
		source := strings.Split(s, "/")
		if len(source) != 2 || source[1] == "" {
			continue
		}
		
		if strings.Compare("edge", source[0]) != 0 {
			continue
		}

		target := msg.GetTarget()
		splitString := strings.Split(target, "@")
		if len(splitString) != 2 || splitString[1] == "" {
			continue
		}
		
		if strings.Compare("device", splitString[0]) != 0 {
			continue
		}
		/*
		* device topic format is :
		* 	$hw/events/device/deviceID/source/target/operation/resource/msgparentid	
		*/
		topic := fmt.Sprintf("$hw/events/device/%s/%s/%s/%s/%s", splitString[1], source[1],
										splitString[0], msg.GetOperation(), msg.GetResource())
		tag := msg.GetTag()
        if tag != "" {
			topic = fmt.Sprintf("%s/%s", topic, tag)
		}else {
			topic = fmt.Sprintf("%s/%s", topic, msg.GetID())
		}

		payload, ok :=msg.GetContent().([]byte)
		if !ok {
			continue
		}
		
		klog.Infof("topic: %s, payload = %s send to device", topic, payload)
		//send to device.
		eb.publish(topic, payload) 
	}
}

func (eb *EventBus) publish(topic string, payload []byte) {
	if eb.conf.MqttMode >= MqttModeBoth {
		// pub msg to external mqtt broker.
		token := eb.MqttClient.PubCli.Publish(topic, 1, false, payload)
		if token.WaitTimeout(TokenWaitTime) && token.Error() != nil {
			klog.Errorf("Error in pubMQTT with topic: %s, %v", topic, token.Error())
		} else {
			klog.Infof("Success in pubMQTT with topic: %s", topic)
		}
	}

	if eb.conf.MqttMode <= MqttModeBoth {
		// pub msg to internal mqtt broker.
		eb.MqttServer.Publish(topic, payload)
	}
}

func (eb *EventBus) subscribe(topic string) {
	if eb.conf.MqttMode >= MqttModeBoth {
		// subscribe topic to external mqtt broker.
		token := eb.MqttClient.SubCli.Subscribe(topic, 1, eb.MqttClient.OnSubMessageReceived)
		if rs, err := mqttBus.CheckClientToken(token); !rs {
			klog.Errorf("Edge-hub-cli subscribe topic: %s, %v", topic, err)
		} 
	}

	if eb.conf.MqttMode <= MqttModeBoth {
		// set topic to internal mqtt broker.
		eb.MqttServer.SetTopic(topic)
	}
}
