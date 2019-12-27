package  switchbus

import (
	"k8s.io/klog"
	"github.com/jwzl/beehive/pkg/core"
	"github.com/jwzl/beehive/pkg/core/context"
)

const (
	ModuleNameSwitchBus= "edge/bus"
)

type SwitchBus struct {
	context		*context.Context
}

// Register this module.
func Register(){	
	sb := &SwitchBus{}
	core.Register(sb)
}

//Name
func (sb *SwitchBus) Name() string {
	return ModuleNameSwitchBus
}

//Group
func (sb *SwitchBus) Group() string {
	return ModuleNameSwitchBus
}

//Start this module.
func (sb *SwitchBus) Start(c *context.Context) {
	klog.Infof("Start the module!")
}

//Cleanup
func (sb *SwitchBus) Cleanup() {
	sb.context.Cleanup(sb.Name())
}
