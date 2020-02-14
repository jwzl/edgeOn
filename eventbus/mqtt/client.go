/*
* mqtt built-in server.
* this is from kubeedge.
*/

package mqtt

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog"
	"crypto/tls"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	
	"github.com/jwzl/edgeOn/common"
	"github.com/jwzl/wssocket/model"
	"github.com/jwzl/beehive/pkg/core/context"	
)

var (
)

// Client struct
type Client struct {
	MQTTUrl string
	PubCli  MQTT.Client
	SubCli  MQTT.Client
	// beehive context.
	Context			*context.Context
}


func NewMqttClient(url string, c *context.Context) *Client {

	return &Client{
		MQTTUrl: url,
		Context: c,	
	}
}

// CheckClientToken checks token is right
func CheckClientToken(token MQTT.Token) (bool, error) {
	if token.Wait() && token.Error() != nil {
		return false, token.Error()
	}
	return true, nil
}

// LoopConnect connect to mqtt server
func (mq *Client) LoopConnect(clientID string, client MQTT.Client) {
	for {
		klog.Infof("start connect to mqtt server with client id: %s", clientID)
		token := client.Connect()
		klog.Infof("client %s isconnected: %v", clientID, client.IsConnected())
		if rs, err := CheckClientToken(token); !rs {
			klog.Errorf("connect error: %v", err)
		} else {
			return
		}
		time.Sleep(5 * time.Second)
	}
}


func (mq *Client) onPubConnectionLost(client MQTT.Client, err error) {
	klog.Errorf("onPubConnectionLost with error: %v", err)
	go mq.InitPubClient()
}

func (mq *Client) onSubConnectionLost(client MQTT.Client, err error) {
	klog.Errorf("onSubConnectionLost with error: %v", err)
	go mq.InitSubClient()
}

func (mq *Client) onSubConnect(client MQTT.Client) {
	for _, t := range SubTopics {
		token := client.Subscribe(t, 1, mq.OnSubMessageReceived)
		if rs, err := CheckClientToken(token); !rs {
			klog.Errorf("edge-hub-cli subscribe topic: %s, %v", t, err)
			return
		}
		klog.Infof("edge-hub-cli subscribe topic to %s", t)
	}
}

// OnSubMessageReceived msg received callback
func (mq *Client) OnSubMessageReceived(client MQTT.Client, message MQTT.Message) {
	// for "$hw/events/twin/#", send to twin
	
	if strings.HasPrefix(message.Topic(), "$hw/events/twin") {
		now := time.Now().UnixNano() / 1e6
	 
		//Header
		msg := model.NewMessage("")
		msg.BuildHeader("", now)

		splitString := strings.Split(message.Topic(), "/")
		//topic format is :$hw/events/twin/deviceID/source/target/operation/resource/msgparentid
		source := splitString[4]
		target := splitString[5]
		operation := splitString[6] 
		resource := splitString[7] 
		//Router
		msg.BuildRouter(source, "", target, resource, operation)	

		if len(splitString) == 9 {
			msg.SetTag(splitString[8])	
		}

		//content
		msg.Content = message.Payload

		klog.Info(fmt.Sprintf("Received msg from mqttserver, deliver to %s with resource %s", common.TwinModuleName, resource))
		mq.Context.Send(common.TwinModuleName, msg)
	}  
}

// HubClientInit create mqtt client config
func (mq *Client) HubClientInit(server, clientID, username, password string) *MQTT.ClientOptions {
	opts := MQTT.NewClientOptions().AddBroker(server).SetClientID(clientID).SetCleanSession(true)
	if username != "" {
		opts.SetUsername(username)
		if password != "" {
			opts.SetPassword(password)
		}
	}
	tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
	opts.SetTLSConfig(tlsConfig)
	return opts
}

// InitSubClient init sub client
func (mq *Client) InitSubClient() {
	timeStr := strconv.FormatInt(time.Now().UnixNano()/1e6, 10)
	right := len(timeStr)
	if right > 10 {
		right = 10
	}

	subID := fmt.Sprintf("hub-client-sub-%s", timeStr[0:right])
	subOpts := mq.HubClientInit(mq.MQTTUrl, subID, "", "")
	subOpts.OnConnect = mq.onSubConnect
	subOpts.AutoReconnect = false
	subOpts.OnConnectionLost = mq.onSubConnectionLost
	mq.SubCli = MQTT.NewClient(subOpts)
	mq.LoopConnect(subID, mq.SubCli)
	klog.Info("finish hub-client sub")
}


// InitPubClient init pub client
func (mq *Client) InitPubClient() {
	timeStr := strconv.FormatInt(time.Now().UnixNano()/1e6, 10)
	right := len(timeStr)
	if right > 10 {
		right = 10
	}

	pubID := fmt.Sprintf("hub-client-pub-%s", timeStr[0:right])
	pubOpts := mq.HubClientInit(mq.MQTTUrl, pubID, "", "")
	pubOpts.OnConnectionLost = mq.onPubConnectionLost
	pubOpts.AutoReconnect = false
	mq.PubCli = MQTT.NewClient(pubOpts)
	mq.LoopConnect(pubID, mq.PubCli)
	klog.Info("finish hub-client pub")
}
