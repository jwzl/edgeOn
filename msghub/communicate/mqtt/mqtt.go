package mqtt

import (
	"fmt"
	"sync"
	"time"
	"strings"
	"k8s.io/klog"
	"github.com/jwzl/mqtt/client"
	"github.com/jwzl/wssocket/fifo"
	"github.com/jwzl/wssocket/model"
	"github.com/jwzl/edgeOn/msghub/config"
)

const (
	//mqtt topic should has the format:
	// mqtt/dgtwin/cloud[edge]/{edgeID}/comm for communication.
	// mqtt/dgtwin/cloud[edge]/{edgeID}/control  for some control message.
	MQTT_SUBTOPIC_PREFIX	= "mqtt/dgtwin/cloud"
	MQTT_PUBTOPIC_PREFIX	= "mqtt/dgtwin/edge"
)

type MqttClient	struct {
	// for mqtt send thread.
	mutex sync.RWMutex
	conf	*config.MqttConfig
	client	*client.Client
	// message fifo.
	messageFifo  *fifo.MessageFifo
}

func NewMqttClient(conf *config.MqttConfig) *MqttClient {
	if conf == nil {
		return nil
	}

	c := client.NewClient(conf.URL, conf.User, conf.Passwd, conf.ClientID)
	if c == nil {
		return nil
	} 
	
	if conf.KeepAliveInterval > 0 {
		c.SetkeepAliveInterval(time.Duration(conf.KeepAliveInterval) * time.Second)
	}
	if conf.PingTimeout	 > 0 {
		c.SetPingTimeout(time.Duration(conf.PingTimeout) * time.Second)
	}
	if conf.QOS >= 0 &&  conf.QOS <= 2 {
		c.SetQOS(byte(conf.QOS))
	}
	c.SetRetain(conf.Retain)
	if conf.MessageCacheDepth > 0 {
		c.SetMessageCacheDepth(conf.MessageCacheDepth) 
	}

	tlsConfig, err := client.CreateTLSConfig(conf.CertFilePath, conf.KeyFilePath)
	if err != nil {
		klog.Infof("TLSConfig Disabled")
	}
	c.SetTlsConfig(tlsConfig)

	return &MqttClient{
		conf: conf,
		client: c,
		messageFifo: fifo.NewMessageFifo(0),
	}
}

func (c *MqttClient) Start() error {
	err := c.client.Start() 
	if err != nil {
		return err
	}

	//TODO: report its edgeID ?

	//Subscribe this topic.
	subTopic := fmt.Sprintf("%s/%s/#", MQTT_SUBTOPIC_PREFIX, c.conf.ClientID)
	err = c.client.Subscribe(subTopic, c.messageArrived)
	if err != nil {
		return err
	}

	return nil
}

func (c *MqttClient) Close(){
	c.client.Close()
}

func (c *MqttClient) messageArrived(topic string, msg *model.Message){
	if msg != nil {
		return
	}

	splitString := strings.Split(topic, "/")
	if len(splitString) != 5 {
		klog.Infof("topic =(%v),  msg ignored", splitString)
		return
	} 
	if strings.Compare(splitString[4], "comm") == 0 {
		// put the model message into fifo.
		c.messageFifo.Write(msg)
	}else{
		//TODO:
	}
}

//ReadMessage read the message from fifo. 
func (c *MqttClient) ReadMessage() (*model.Message, error){
	return c.messageFifo.Read()
}

//WriteMessage publish the message to cloud.
func (c *MqttClient) WriteMessage(msg *model.Message) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	pubTopic := fmt.Sprintf("%s/%s/comm", MQTT_PUBTOPIC_PREFIX, c.conf.ClientID)
	return c.client.Publish(pubTopic, msg)
}
