package  eventbus

import (
	"k8s.io/klog"
	"github.com/jwzl/edgeOn/common"
	"github.com/jwzl/beehive/pkg/core"
	"github.com/jwzl/beehive/pkg/core/context"

	mqttBus "github.com/jwzl/edgeOn/eventbus/mqtt"
)

const (
)

type EventBus struct {
	context		*context.Context
}

// Register this module.
func Register(){	
	sb := &EventBus{}
	core.Register(sb)
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
}

//Cleanup
func (eb *EventBus) Cleanup() {
	eb.context.Cleanup(eb.Name())
}
