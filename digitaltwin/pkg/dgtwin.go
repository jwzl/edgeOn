package digitaltwin

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/context"
)

const (
	ModuleNameDTwin= "dtwin"
)

type DGTwinModule struct {
	context 	*context.Context
	controller	*DGTwinController
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
	dtm.controller = controller.NewDGTwinController(c)
}

//Cleanup
func (dtm *DGTwinModule) Cleanup() {
	dtm.context.Cleanup(dtm.Name())
}
