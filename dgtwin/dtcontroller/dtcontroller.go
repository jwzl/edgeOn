package dtcontroller

import (
	"time"
	"errors"
	"strings"	
	"k8s.io/klog"
	"github.com/jwzl/wssocket/model"
	"github.com/jwzl/beehive/pkg/core/context"
	"github.com/jwzl/edgeOn/dgtwin/types"
	"github.com/jwzl/edgeOn/dgtwin/dtcontext"
	"github.com/jwzl/edgeOn/dgtwin/dtmodule"
)


type DGTwinController struct {
	ID			string
	Stop		chan bool
	context 	*dtcontext.DTContext
}

// NewDGTwinController: Create digital twin controller.
func NewDGTwinController(id	string, c *context.Context) *DGTwinController {
	ctx := dtcontext.NewDTContext(c)
	if ctx == nil {
		return nil
	}
	stop := make(chan bool, 1)

	// create and register all modules.
	modules := []string{types.DGTWINS_MODULE_COMM, types.DGTWINS_MODULE_TWINS, types.DGTWINS_MODULE_PROPERTY}
	for _, name := range modules {
		dtm := dtmodule.NewDTModule(name)
		ctx.RegisterDTModule(dtm)
	}

	dtc := &DGTwinController{
		ID:		 id,
		Stop:	 stop,
		context: ctx,	
	}
	return dtc
}

func (dtc *DGTwinController) Start() error {
	//Start all sub-modules.
	for _ , module := range dtc.context.Modules {
		go module.Start()
	}

	//Start a goroutine to recieve message.
	go dtc.RecvModuleMsg()

	//loop for sub-module's health check and stop modules.	
	for{
		select {
		case <-time.After(60*time.Second):
			//health check.
			for name , module := range dtc.context.Modules {
				v, exist := dtc.context.ModuleHealth.Load(name)
				if exist {
					now := time.Now().Unix()
					if now - v.(int64) > 80 {
						klog.Infof("%s module is not healthy, we restart it", name)
						go module.Start()
					}
				}
		
				//ping the module.
				if ch, exist := dtc.context.HeartBeatChan[name]; exist {
					ch <- "ping"
				}
			}
		case <-dtc.Stop:
			// Stop all sub-modules.
			for name , _ := range dtc.context.Modules {
				dtc.context.StopModule(name) 
			}

			return nil
		}
	}
}

func (dtc *DGTwinController) CleanUp(){
	for name , _ := range dtc.context.Modules {
		dtc.context.StopModule(name) 
	}
}

func (dtc *DGTwinController) RecvModuleMsg(){
	for {
		// Recieve the message from other modules.
		if v, err := dtc.context.Receive(); err== nil {
			msg, isMsgType := v.(*model.Message)
			if isMsgType {
				klog.Infof("digital twin message arrived!")
				err = dtc.dispatch(msg)
				if err != nil {
					klog.Infof("dispatch err (%v), Ignored!", err)
				}	
			} 
		}
	}
}

// message dispatch.
func (dtc *DGTwinController) dispatch(msg *model.Message) error {
	if msg == nil {
		return errors.New("message is nil")
	}

	target   := msg.GetTarget()
	resource := msg.GetResource()
	
	if strings.Compare(types.MODULE_NAME, target) != 0 {
		return errors.New("message is not to this module ")
	}

	if strings.Contains(resource, types.DGTWINS_MODULE_TWINS){
		dtc.context.SendToModule(types.DGTWINS_MODULE_TWINS, msg)
	}else if strings.Contains(resource, types.DGTWINS_MODULE_PROPERTY) {
		dtc.context.SendToModule(types.DGTWINS_MODULE_PROPERTY, msg)
	}
	return nil
}
