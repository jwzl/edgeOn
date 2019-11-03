package dtcontext

import (
	"time"
	"sync"
	"errors"
	"strings"
	"k8s.io/klog"
	"github.com/jwzl/wssocket/model"
	"github.com/jwzl/beehive/pkg/core/context"
	"github.com/jwzl/edgeOn/digitaltwin/pkg/types"
)

type DTContext struct {
	DeviceID		string
	Context			*context.Context
	Modules			map[string]DTModule
	CommChan		map[string]chan interface{}
	HeartBeatChan 	map[string]chan interface{}
	ConfirmChan		chan interface{}

	ModuleHealth	*sync.Map
	MessageCache	*sync.Map
	//this is for watch event.
	WatchCache	*sync.Map
	// Cache for digitaltwin	
	DGTwinList	*sync.Map
	DGTwinMutex	*sync.Map	
}

func NewDTContext(c *context.Context) *DTContext {
	modules	:= make(map[string]DTModule)
	commChan := make(map[string]chan interface{})
	heartBeatChan:= make(map[string]chan interface{})
	confirmChan :=	make(chan interface{})
	var modulesHealth sync.Map
	var messageCache sync.Map
	var dgTwinList sync.Map
	var dgTwinMutex sync.Map

	return &DTContext{
		Context:	c,
		Modules:	modules,
		CommChan:	commChan,
		HeartBeatChan:	heartBeatChan,
		ConfirmChan:	confirmChan,
		ModuleHealth:	&modulesHealth,
		MessageCache:   &messageCache,
		DGTwinList: 	&dgTwinList,
		DGTwinMutex:	&dgTwinMutex,
	}
}

func (dtc *DTContext) RegisterDTModule(dtm DTModule){
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
	mutex, ok := dtc.GetMutex(deviceID)
	if ok {
		mutex.Lock()
		return true
	}

	return false
}

//unlock device by ID 
func (dtc *DTContext) Unlock (deviceID string) bool {
	mutex, ok := dtc.GetMutex(deviceID)
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

	_, isDGTwin := v.(*types.DigitalTwin)
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
	msg.BuildRouter(source, "", target, resource, operation)
	
	//content
	msg.Content = content
	
	return msg
}

//send message to module.
func (dtc *DTContext) Send(module string, msg *model.Message) {
	dtc.Context.Send(module, *msg)
}

//SendResponseMessage Send Response conten.
func (dtc *DTContext) SendResponseMessage(requestMsg *model.Message, content []byte){
	target := requestMsg.GetSource()
	resource := requestMsg.GetResource()

	modelMsg := dtc.BuildModelMessage(types.MODULE_NAME, target, 
					types.DGTWINS_OPS_RESPONSE, resource, content)	
	modelMsg.SetTag(requestMsg.GetID())	
	klog.Infof("Send response message (%v)", modelMsg)

	dtc.SendToModule(types.DGTWINS_MODULE_COMM, modelMsg)
}

//SendTwinMessage2Device Send twin message to device.
func (dtc *DTContext) SendTwinMessage2Device(requestMsg *model.Message, action string, twins []*types.DigitalTwin) error {
	resource := requestMsg.GetResource()

	msgContent, err := types.BuildTwinMessage(action, twins)
	if err == nil {
		modelMsg := dtc.BuildModelMessage(types.MODULE_NAME, "device", 
												action, resource, msgContent)
		klog.Infof("Send device message (%v) ", modelMsg)
		dtc.SendToModule(types.DGTWINS_MODULE_COMM, modelMsg)
		
		return nil
	}
	
	return err
}

func (dtc *DTContext) UpdateWatchCache(we *types.WatchEvent) {

	if we == nil {
		return 
	}

	if dtc.WatchCache == nil {
		var watchCache	sync.Map
		dtc.WatchCache = &watchCache
	}

	v, exist := dtc.WatchCache.Load(we.TwinID)
	if !exist {
		dtc.ModuleHealth.Store(we.TwinID, we)
	}else{
		watchEvent, isThisType := v.(*types.WatchEvent)
		if isThisType {
			for _, value :=  range we.List
				ok := false 	
				for _, val := range watchEvent.List {
					if 	value == val {
						ok = true
						break
					}					
				}
				if ok {
					continue
				}
				append(watchEvent.List, value)
			}
		}
	}
}
