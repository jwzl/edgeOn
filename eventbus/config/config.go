package config

import (
	"k8s.io/klog"
	"github.com/jwzl/beehive/pkg/common/config"
)


// EventBus indicates the event bus module config
type EventBusConfig struct {
	// MqttQOS indicates mqtt qos
	// 0: QOSAtMostOnce, 1: QOSAtLeastOnce, 2: QOSExactlyOnce
	// default 0
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	MqttQOS int `json:"mqttQOS"`
	// MqttRetain indicates whether server will store the message and can be delivered to future subscribers
	// if this flag set true, sever will store the message and can be delivered to future subscribers
	// default false
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	MqttRetain bool `json:"mqttRetain"`
	// MqttSessionQueueSize indicates the size of how many sessions will be handled.
	// default 100
	MqttSessionQueueSize int `json:"mqttSessionQueueSize,omitempty"`
	// MqttServerInternal indicates internal mqtt broker url
	// default tcp://127.0.0.1:1884
	MqttServerInternal string `json:"mqttServerInternal,omitempty"`
	// MqttServerExternal indicates external mqtt broker url
	// default tcp://127.0.0.1:1883
	MqttServerExternal string `json:"mqttServerExternal,omitempty"`
	// MqttMode indicates which broker type will be choose
	// 0: internal mqtt broker enable only. 1: internal and external mqtt broker enable. 2: external mqtt broker enable only
	// +Required
	// default: 0
	MqttMode int `json:"mqttMode"`
}

func GetEventBusConfig() *EventBusConfig {
	eBConfig := &EventBusConfig{}

	qos, err := config.CONFIG.GetValue("eventbus.mqtt.qos").ToInt()
	if err != nil {
		klog.Infof("eventbus.mqtt.qos is empty")
		qos = 2
	}
	eBConfig.MqttQOS = qos

	retain, err := config.CONFIG.GetValue("eventbus.mqtt.retain").ToBool()
	if err != nil {
		klog.Infof("eventbus.mqtt.retain is empty")
		retain = false
	}
	eBConfig.MqttRetain = retain

	sessionQueueSize, err := config.CONFIG.GetValue("eventbus.mqtt.session-queue-size").ToInt()
	if err != nil {
		klog.Infof("eventbus.mqtt.session-queue-size is empty")
		sessionQueueSize = 100
	}
	eBConfig.MqttSessionQueueSize = sessionQueueSize

	MqttServerInternal, err := config.CONFIG.GetValue("eventbus.mqtt.mqttServerInternal").ToString()
	if err != nil {
		klog.Infof("eventbus.mqtt.mqttServerInternal is empty")
		MqttServerInternal = "tcp://127.0.0.1:1884"
	}
	eBConfig.MqttServerInternal = MqttServerInternal

	MqttServerExternal, err := config.CONFIG.GetValue("eventbus.mqtt.mqttServerExternal").ToString()
	if err != nil {
		klog.Infof("eventbus.mqtt.mqttServerExternal is empty")
		MqttServerExternal = "tcp://127.0.0.1:1883"
	}
	eBConfig.MqttServerExternal = MqttServerExternal

	mode, err := config.CONFIG.GetValue("eventbus.mqtt.mode").ToInt()
	if err != nil {
		klog.Infof("eventbus.mqtt.mode is empty")
		mode = 0
	}
	eBConfig.MqttMode = mode

	return eBConfig
} 
