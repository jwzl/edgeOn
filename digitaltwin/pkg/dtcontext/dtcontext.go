package dtcontext

import (
	"errors"
	"k8s.io/klog"
	"github.com/kubeedge/beehive/pkg/core/context"
)

type DTContext struct {
	DeviceID	string
	Context		*context.Context
	Modules		map[string]dtmodule.DTModule
	CommChan	map[string]chan interface{}
	HeartBeatChan map[string]chan interface{}
	ConfirmChan	chan interface{}	
}

func NewDTContext(c *context.Context) *DTContext {
	modules	:= make(map[string]dtmodule.DTModule)
	commChan := make(map[string]chan interface{})
	heartBeatChan:= make(map[string]chan interface{})
	confirmChan :=	make(chan interface{})

	return &DTContext{
		Context:	c,
		Modules:	modules,
		CommChan:	commChan,
		HeartBeatChan:	heartBeatChan,
		ConfirmChan:	confirmChan,
	}
}

func (dtc *DTContext) RegisterDTModule(dtm dtmodule.DTModule){
	moduleName := dtm.Name()
	dtc.CommChan[moduleName] = make(chan interface{}, 128)
	dtc.HeartBeatChan[moduleName] = make(chan interface{}, 128)
	//Pass dtcontext to dtmodule.
	dtm.InitModule(dtc)
	dtc.Modules[moduleName] = dtm
}

func (dtc *DTContext) SendToModule(dtmName string, content interface{}) error {
	if ch, exist := dtc.CommChan[dtmName];  exist {
		ch <- content
		return nil
	}

	return errors.New("Channel not found")
}
