package msghub

import (
	"k8s.io/klog"
	"github.com/jwzl/beehive/pkg/core"
	"github.com/jwzl/beehive/pkg/core/context"
)

const (
	ModuleNameHub= "edge/hub"
)

type MsgHub struct {
	context		*context.Context
	mhc		*Controller
}

// Register this module.
func Register(){	
	mh := &MsgHub{}
	core.Register(mh)
}

//Name
func (mh *MsgHub) Name() string {
	return ModuleNameHub
}

//Group
func (mh *MsgHub) Group() string {
	return ModuleNameHub
}

//Start this module.
func (mh *MsgHub) Start(c *context.Context) {
	klog.Infof("Start the module!")
	mh.mhc = NewController(c)

	// Start the message hub.
	mh.mhc.Start()
}

//Cleanup
func (mh *MsgHub) Cleanup() {
	mh.context.Cleanup(mh.Name())
}
