package dgtwin

import (
	_"k8s.io/klog"
	"github.com/jwzl/beehive/pkg/core"
	"github.com/jwzl/beehive/pkg/core/context"
	"github.com/jwzl/edgeOn/dgtwin/pkg/dtcontroller"
)

const (
	ModuleNameDTwin= "dtwin"
)

type DGTwinModule struct {
	context 	*context.Context
	controller	*dtcontroller.DGTwinController
}

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
	dtm.context = c
	dtm.controller = dtcontroller.NewDGTwinController("", c)
}

//Cleanup
func (dtm *DGTwinModule) Cleanup() {
	dtm.controller.CleanUp()
	dtm.context.Cleanup(dtm.Name())
}
