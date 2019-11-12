package mqtt

import (
	"fmt"
	"time"
	"strings"
	"k8s.io/klog"
	"github.com/jwzl/mqtt/client"
	"github.com/jwzl/wssocket/fifo"
	"github.com/jwzl/wssocket/model"
)

const (
	//mqtt topic should has the format:
	// mqtt/dgtwin/cloud[edge]/{edgeID}/comm for communication.
	// mqtt/dgtwin/cloud[edge]/{edgeID}/control  for some control message.
	MQTT_SUBTOPIC_PREFIX	= "mqtt/dgtwin/cloud"
	MQTT_PUBTOPIC_PREFIX	= "mqtt/dgtwin/edge"
)

type MqttClient	struct {
	config	*MqttConfig
	client	*client.Client
	// message fifo.
	messageFifo  *fifo.MessageFifo
}

type MqttConfig struct {
	URL				string
	ClientID		string
	User			string
	Passwd			string
	CertFilePath    string
	KeyFilePath     string
	keepAliveInterval	int
	PingTimeout		int  
}

func NewMqttClient(conf *MqttConfig) *MqttClient {
	if conf == nil {
		return nil
	}

	c := client.NewClient(conf.URL, conf.User, conf.Passwd, conf.ClientID)
	if c == nil {
		return nil
	} 
	
	if conf.keepAliveInterval > 0 {
		c.SetkeepAliveInterval(time.Duration(conf.keepAliveInterval) * time.Second)
	}
	if conf.PingTimeout	 > 0 {
		c.SetPingTimeout(time.Duration(conf.PingTimeout) * time.Second)
	}
	tlsConfig, err := client.CreateTLSConfig(conf.CertFilePath, conf.KeyFilePath)
	if err != nil {
		klog.Infof("TLSConfig Disabled")
	}
	c.SetTlsConfig(tlsConfig)

	return &MqttClient{
		config: conf,
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
	subTopic := fmt.Sprintf("%s/%s/#", MQTT_SUBTOPIC_PREFIX, c.config.ClientID)
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
	pubTopic := fmt.Sprintf("%s/%s/comm", MQTT_PUBTOPIC_PREFIX, c.config.ClientID)
	return c.client.Publish(pubTopic, msg)
}
