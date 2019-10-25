package dtcontext

import (
	"fmt"
	"time"
	"sync"
	"errors"
	"k8s.io/klog"
	"github.com/kubeedge/beehive/pkg/core/context"
)

type DTContext struct {
	DeviceID		string
	Context			*context.Context
	Modules			map[string]dtmodule.DTModule
	CommChan		map[string]chan interface{}
	HeartBeatChan 	map[string]chan interface{}
	ConfirmChan		chan interface{}

	ModuleHealth	*sync.Map
	MessageCache	*sync.Map
	// Cache for digitaltwin	
	DGTwinList	*sync.Map
	DGTwinMutex	*sync.Map	
}

func NewDTContext(c *context.Context) *DTContext {
	modules	:= make(map[string]dtmodule.DTModule)
	commChan := make(map[string]chan interface{})
	heartBeatChan:= make(map[string]chan interface{})
	confirmChan :=	make(chan interface{})
	var modulesHealth sync.Map
	var messageCache sync.Map

	return &DTContext{
		Context:	c,
		Modules:	modules,
		CommChan:	commChan,
		HeartBeatChan:	heartBeatChan,
		ConfirmChan:	confirmChan,
		ModuleHealth:	&modulesHealth,
		MessageCache:   &messageCache,
	}
}

func (dtc *DTContext) RegisterDTModule(dtm dtmodule.DTModule){
	moduleName := dtm.Name()
	dtc.CommChan[moduleName] = make(chan interface{}, 128)
	dtc.HeartBeatChan[moduleName] = make(chan interface{}, 128)
	//Pass dtcontext to dtmodule.
	dtm.InitModule(dtc, dtc.CommChan[moduleName], dtc.HeartBeatChan[moduleName], dtc.ConfirmChan)
	dtc.Modules[moduleName] = dtm
}

// send msg  to sub-module
func (dtc *DTContext) SendToModule(dtmName string, content interface{}) error {
	if ch, exist := dtc.CommChan[dtmName];  exist {
		ch <- content
		return nil
	}

	return errors.New("Channel not found")
}

//handle heartbeat.
func (dtc *DTContext) HandleHeartBeat(dtmName string, content string) error {
	if strings.Compare(content, "ping")	== 0 {
		dtc.ModuleHealth.Store(dtmName, time.Now().Unix())
		klog.Infof("%s is healthy %v", dtmName, time.Now().Unix())
	}else if strings.Compare(content, "stop")	== 0 {
		klog.Infof("%s stop", dtmName)
		return errors.New("stop")
	}

	return nil
}

//GetMutex get the device mutex
func (dtc *DTContext) GetMutex (deviceID string) (*sync.Mutex, bool) {
	v, exist := dtc.DGTwinMutex.Load(deviceID)
	if !exist {
		return nil, false
	}

	mutex, isMutex := v.(*sync.Mutex)
	if !isMutex {
		return nil, false
	}

	return mutex, true
}

//Lock  device by ID
func (dtc *DTContext) Lock (deviceID string) bool {
	mutex, ok := GetMutex(deviceID)
	if ok {
		mutex.Lock()
		return true
	}

	return false
}

//unlock device by ID 
func (dtc *DTContext) Unlock (deviceID string) bool {
	mutex, ok := GetMutex(deviceID)
	if ok {
		mutex.Unlock()
		return true
	}

	return false
}

//digital twin is exist.
func (dtc *DTContext) DGTwinIsExist (deviceID string) bool {
	v, exist := dtc.DGTwinList.Load(deviceID)
	if !exist {
		return false
	}

	dgtwin, isDGTwin := v.(*types.DigitalTwin)
	if !isDGTwin {
		return false
	}

	return true
}

func (dtc *DTContext) BuildModelMessage(source string, target string, operation string, resource string, content interface{}) *model.Message {
	now := time.Now().UnixNano() / 1e6
	
	//Header
	msg := model.NewMessage("")
	msg.BuildHeader("", now)

	//Router
	BuildRouter(source, "", target, resource, operation)
	
	//content
	msg.Content = content
	
	return msg
}

//send message to module.
func (dtc *DTContext) Send(module string, msg *model.Message) {
	dtc.Context.Send(module, *msg)
}

func (dtc *DTContext) BuildDTMessage(source string, target string, operation string, resource string, msg *model.Message) *types.DTMessage {
	dtMsg := &types.DTMessage{
		Msg: 	msg,
		Source: source,
		Target: target,
		Operation: operation,
		Resource:  resource, 
	}
	
	return dtMsg
}
