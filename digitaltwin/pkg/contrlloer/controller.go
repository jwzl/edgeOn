package controller

import (
	"k8s.io/klog"
)


type DGTwinController struct {
	context 	*context.Context
}

func NewDGTwinController(c *context.Context) *DGTwinController {
	dtc := &DGTwinController{
		context: c,
	}


	return dtc
}

func (dtc *DGTwinController) Start() {

	go RecvModuleMsg()
}

func (dtc *DGTwinController) RecvModuleMsg(){
	for {
		// Recieve the message from other modules.
		if msg, err := dtc.context.Receive("dtwin"); err== nil {
			klog.Infof("dtwin message arrived!")
			dtMsg := types.DTMessage{Msg: &msg}
		}
	}
}

func (dtc *DGTwinController) classifyMsg(message *types.DTMessage){
	message.Msg.
}
