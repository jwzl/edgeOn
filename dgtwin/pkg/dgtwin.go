package dgtwin

import (
	"k8s.io/klog"
	"github.com/jwzl/beehive/pkg/core"
	"github.com/jwzl/beehive/pkg/core/context"
	"github.com/jwzl/edgeOn/dgtwin/pkg/dtcontroller"
)

const (
	ModuleNameDTwin= "edge/dgtwin"
)

type DGTwinModule struct {
	context		*context.Context
	controller	*dtcontroller.DGTwinController
}

// Register this module.
func Register(){	
	dtm := &DGTwinModule{}
	core.Register(dtm)
}

//Name
func (dtm *DGTwinModule) Name() string {
	return ModuleNameDTwin
}

//Group
func (dtm *DGTwinModule) Group() string {
	return ModuleNameDTwin
}

//Start this module.
func (dtm *DGTwinModule) Start(c *context.Context) {
	if c == nil {
		klog.Errorf("context is nil")
		return 
	}

	dtm.context = c
	dtm.controller = dtcontroller.NewDGTwinController("", c)
	if dtm.controller == nil {
		klog.Errorf("Create twin controller failed.")
		return 	
	}

	klog.Infof("Start digital twin module.")
	err := dtm.controller.Start()
	if  err != nil {
		klog.Errorf("Start twin controller failed (%v).", err)
	}
}

//Cleanup
func (dtm *DGTwinModule) Cleanup() {
	dtm.controller.CleanUp()
	dtm.context.Cleanup(dtm.Name())
}
