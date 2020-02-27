/*
* mqtt built-in server.
* this is from kubeedge.
*/

package mqtt

import (
	"fmt"
	"time"
	"strings"
	"k8s.io/klog"
	
	"github.com/jwzl/wssocket/model"
	"github.com/256dpi/gomqtt/broker"
	"github.com/256dpi/gomqtt/packet"
	"github.com/256dpi/gomqtt/topic"
	"github.com/256dpi/gomqtt/transport"

	"github.com/jwzl/edgeOn/common"
	"github.com/jwzl/beehive/pkg/core/context"
)

var (
	// SubTopics which edge-client should be sub
	SubTopics = []string{
		//"$hw/events/upload/#",
		"$hw/events/twin/#",
	}
)
//Server serve as an internal mqtt broker.
type Server struct {
	// Internal mqtt url
	url string

	// Used to save and match topic, it is thread-safe tree.
	tree *topic.Tree

	// A server accepts incoming connections.
	server transport.Server

	// A MemoryBackend stores all in memory.
	backend *broker.MemoryBackend

	// Qos has three types: QOSAtMostOnce, QOSAtLeastOnce, QOSExactlyOnce.
	// now we use QOSAtMostOnce as default.
	qos int

	// If set retain to true, the topic message will be saved in memory and
	// the future subscribers will receive the msg whose subscriptions match
	// its topic.
	// If set retain to false, then will do nothing.
	retain bool

	// A sessionQueueSize will default to 100
	sessionQueueSize int

	// beehive context.
	Context			*context.Context
}


// NewMqttServer create an internal mqtt server.
func NewMqttServer(sqz int, url string, retain bool, qos int, c *context.Context) *Server {
	return &Server{
		sessionQueueSize: sqz,
		url:              url,
		tree:             topic.NewStandardTree(),
		retain:           retain,
		qos:              qos,
		Context:		  c,
	}
}

// Run launch a server and accept connections.
func (m *Server) Run() error {
	var err error

	m.server, err = transport.Launch(m.url)
	if err != nil {
		klog.Errorf("Launch transport failed %v", err)
		return err
	}

	m.backend = broker.NewMemoryBackend()
	m.backend.SessionQueueSize = m.sessionQueueSize

	m.backend.Logger = func(event broker.LogEvent, client *broker.Client, pkt packet.Generic, msg *packet.Message, err error) {
		if event == broker.MessagePublished {
			if len(m.tree.Match(msg.Topic)) > 0 {
				m.onSubscribe(msg)
			}
		}
	}

	engine := broker.NewEngine(m.backend)
	engine.Accept(m.server)

	return nil
}

// onSubscribe will be called if the topic is matched in topic tree.
func (m *Server) onSubscribe(message *packet.Message) {
	// for "$hw/events/twin/#", send to twin
	
	if strings.HasPrefix(message.Topic, "$hw/events/twin") {
		now := time.Now().UnixNano() / 1e6
	 
		//Header
		msg := model.NewMessage("")
		msg.BuildHeader("", now)

		splitString := strings.Split(message.Topic, "/")
		//topic format is :$hw/events/twin/deviceID/source/target/operation/resource/msgparentid
		source := splitString[4]
		target := splitString[5]
		operation := splitString[6] 
		resource := splitString[7] 
		//Router
		msg.BuildRouter(source, "", "edge/"+target, resource, operation)	

		if len(splitString) == 9 {
			msg.SetTag(splitString[8])	
		}

		//content
		msg.Content = message.Payload

		klog.Info(fmt.Sprintf("Received msg from mqttserver, deliver to %s with resource %s", common.TwinModuleName, resource))
		m.Context.Send(common.TwinModuleName, msg)
	}  
}

// InitInternalTopics sets internal topics to server by default.
func (m *Server) InitInternalTopics() {
	for _, v := range SubTopics {
		m.tree.Set(v, packet.Subscription{Topic: v, QOS: packet.QOS(m.qos)})
		klog.Infof("Subscribe internal topic to %s", v)
	}
}

// SetTopic set the topic to internal mqtt broker.
func (m *Server) SetTopic(topic string) {
	m.tree.Set(topic, packet.Subscription{Topic: topic, QOS: packet.QOSAtMostOnce})
}

// Publish will dispatch topic msg to its subscribers directly.
func (m *Server) Publish(topic string, payload []byte) {
	client := &broker.Client{}

	msg := &packet.Message{
		Topic:   topic,
		Retain:  m.retain,
		Payload: payload,
		QOS:     packet.QOS(m.qos),
	}
	m.backend.Publish(client, msg, nil)
}
