package config

import (
	"k8s.io/klog"
	"github.com/jwzl/beehive/pkg/common/config"
)

type MqttConfig struct {
	URL				string
	ClientID		string
	User			string
	Passwd			string
	CertFilePath    string
	KeyFilePath     string
	KeepAliveInterval	int
	PingTimeout			int  
	QOS				 	int
	Retain			   	bool	
	MessageCacheDepth  	uint
}

func GetMqttConfig() (*MqttConfig, error) {
	conf := &MqttConfig{}
	
	url, err := config.CONFIG.GetValue("msghub.mqtt.broker").ToString()
	if err != nil {
		klog.Errorf("Failed to get broker url for mqtt client: %v", err)
		return nil, err
	}
	conf.URL = url

	id, err := config.CONFIG.GetValue("dgtwin.id").ToString()
	if err != nil {
		klog.Warningf("Failed to get client id: %v", err)
		return nil, err
	}
	conf.ClientID = id	

	user, err := config.CONFIG.GetValue("msghub.mqtt.user").ToString()
	if err != nil {
		klog.Infof("msghub.mqtt.user is empty")
		user = ""
	}
	conf.User = user

	passwd, err := config.CONFIG.GetValue("msghub.mqtt.passwd").ToString()
	if err != nil {
		klog.Infof("msghub.mqtt.passwd is empty")
		passwd = ""
	}
	conf.Passwd = passwd

	certfile, err := config.CONFIG.GetValue("msghub.mqtt.certfile").ToString()
	if err != nil {
		klog.Infof("msghub.mqtt.certfile is empty")
		certfile = ""
	}
	conf.CertFilePath = certfile

	keyfile, err := config.CONFIG.GetValue("msghub.mqtt.keyfile").ToString()
	if err != nil {
		klog.Infof("msghub.mqtt.keyfile is empty")
		keyfile = ""
	}
	conf.KeyFilePath = keyfile
	
	keepAliveInterval, err := config.CONFIG.GetValue("msghub.mqtt.keep-alive-interval").ToInt()
	if err != nil {
		klog.Infof("msghub.mqtt.keep-alive-interval is empty")
		keepAliveInterval = 120
	}
	conf.KeepAliveInterval = keepAliveInterval

	pingTimeout, err := config.CONFIG.GetValue("msghub.mqtt.ping-timeout").ToInt()
	if err != nil {
		klog.Infof("msghub.mqtt.ping-timeout is empty")
		pingTimeout = 120
	}
	conf.PingTimeout = pingTimeout

	qos, err := config.CONFIG.GetValue("msghub.mqtt.qos").ToInt()
	if err != nil {
		klog.Infof("msghub.mqtt.qos is empty")
		qos = 2
	}
	conf.QOS = qos

	retain, err := config.CONFIG.GetValue("msghub.mqtt.retain").ToBool()
	if err != nil {
		klog.Infof("msghub.mqtt.retain is empty")
		retain = false
	}
	conf.Retain = retain

	sessionQueueSize, err := config.CONFIG.GetValue("msghub.mqtt.session-queue-size").ToInt()
	if err != nil {
		klog.Infof("msghub.mqtt.session-queue-size is empty")
		sessionQueueSize = 100
	}
	conf.MessageCacheDepth = uint(sessionQueueSize)

	return conf, nil
}

type WebsocketServerConfig struct{
	URL				string
	EdgeID			string
	CertFilePath    string
	KeyFilePath     string
	HandshakeTimeout   int
	ReadDeadline       int
	WriteDeadline      int
}

func GetWSServerConfig() (*WebsocketServerConfig, error) {
	conf := &WebsocketServerConfig{}

	url, err := config.CONFIG.GetValue("msghub.websocket.url").ToString()
	if err != nil {
		klog.Errorf("Failed to get url for websocket server: %v", err)
		return nil, err
	}
	conf.URL = url

	id, err := config.CONFIG.GetValue("dgtwin.id").ToString()
	if err != nil {
		klog.Warningf("Failed to get edge id: %v", err)
		return nil, err
	}
	conf.EdgeID = id

	certfile, err := config.CONFIG.GetValue("msghub.websocket.certfile").ToString()
	if err != nil {
		klog.Infof("msghub.websocket.certfile is empty")
		certfile = ""
	}
	conf.CertFilePath = certfile

	keyfile, err := config.CONFIG.GetValue("msghub.websocket.keyfile").ToString()
	if err != nil {
		klog.Infof("msghub.websocket.keyfile is empty")
		keyfile = ""
	}
	conf.KeyFilePath = keyfile

	handshakeTimeout, err := config.CONFIG.GetValue("msghub.websocket.handshake-timeout").ToInt()
	if err != nil {
		klog.Infof("msghub.websocket.keyfile is empty")
		handshakeTimeout = 30
	}
	conf.HandshakeTimeout = handshakeTimeout

	readDeadline, err := config.CONFIG.GetValue("msghub.websocket.read-deadline").ToInt()
	if err != nil {
		klog.Infof("msghub.websocket.read-deadline is empty")
		readDeadline = 15
	}
	conf.ReadDeadline = readDeadline

	writeDeadline, err := config.CONFIG.GetValue("msghub.websocket.write-deadline").ToInt()
	if err != nil {
		klog.Infof("msghub.websocket.write-deadline is empty")
		writeDeadline = 15
	}
	conf.WriteDeadline = writeDeadline

	return conf, nil
}
