package dtmodule

import (
	"time"
	"strings"
	"k8s.io/klog"
	"github.com/jwzl/edgeOn/common"
	"github.com/jwzl/wssocket/model"
	"github.com/jwzl/edgeOn/dgtwin/types"
	"github.com/jwzl/edgeOn/dgtwin/dtcontext"
)

type CommandFunc  func(msg interface{}) error

type CommModule struct {
	name	string
	context			*dtcontext.DTContext
	//for msg communication
	recieveChan		chan interface{}
	// for module's health check.
	heartBeatChan	chan interface{}
	confirmChan		chan interface{}
	CommandTbl 	map[string]CommandFunc
}

func NewCommModule() *CommModule {
	return &CommModule{name: types.DGTWINS_MODULE_COMM}
}

func (cm *CommModule) Name() string {
	return cm.name
}

//Init the comm module.
func (cm *CommModule) InitModule(dtc *dtcontext.DTContext, comm, heartBeat, confirm chan interface{}) {
	cm.context = dtc
	cm.recieveChan = comm
	cm.heartBeatChan = heartBeat
	cm.confirmChan = confirm
	//cm.initDeviceCommandTable()
}

//Start comm module
//TODO: Device should has a healthcheck.
func (cm *CommModule) Start(){
	//Start loop.
	checkTimeoutCh := time.After(2*time.Second)
	for {
		select {
		case msg, ok := <-cm.recieveChan:
			if !ok {
				//channel closed.
				return
			}
			
			message, isMsgType := msg.(*model.Message )
			if isMsgType {
		 		// do handle.
				target := message.GetTarget()
				if strings.Contains(target, common.DeviceName) {
					//send to device.
					klog.Infof("send to device")	
					cm.sendMessageToDevice(message) 	
				}else if strings.Contains(target, common.CloudName) {
					//send to message cloud.
					klog.Infof("send to cloud")
					cm.sendMessageToHub(message)	
				}else if strings.Contains(target, "edge") {
					if strings.Contains(target, types.MODULE_NAME) {
						//this is response or internal communication.
						cm.dealMessageToTwin(message)
					}else {
						//this is edge/app
						klog.Infof("send to edge/app")
						cm.sendMessageToHub(message)
					}
				}else{
					klog.Warningf("error message format, Ignore (%v)", message)
				}				
			}
		case v, ok := <-cm.heartBeatChan:
			if !ok {
				return
			}
			
			err := cm.context.HandleHeartBeat(cm.Name(), v.(string))
			if err != nil {
				klog.Infof("%s module stopped", cm.Name())
				return
			}
		case <-checkTimeoutCh:
			//check  the MessageCache for response.
			cm.dealMessageTimeout()	
			checkTimeoutCh = time.After(2*time.Second)
		}
	}
}

// sendMessageToDevice
func (cm *CommModule) sendMessageToDevice(msg *model.Message) {
	operation := msg.GetOperation()

	if strings.Compare(common.DGTWINS_OPS_RESPONSE, operation) != 0 {
		//cache this message for confirm recieve the response.
		id := msg.GetID() 
		_, exist := cm.context.MessageCache.Load(id)
		if !exist {	
			cm.context.MessageCache.Store(id, msg)
		}
	}

	//send message to protocol bus.
	cm.context.Send(common.BusModuleName, msg)
}

//sendMessageToHub
func (cm *CommModule) sendMessageToHub(msg *model.Message) {
	operation := msg.GetOperation()

	if strings.Compare(common.DGTWINS_OPS_RESPONSE, operation) != 0 {
		//cache this message for confirm recieve the response.
		id := msg.GetID() 
		_, exist := cm.context.MessageCache.Load(id)
		if !exist {
			cm.context.MessageCache.Store(id, msg)
		}
	}
	//send message to message hub.
	cm.context.Send(common.HubModuleName, msg)
}

//dealMessageToTwin
func (cm *CommModule) dealMessageToTwin(msg *model.Message) {
	//If we recieve the response message, then delete cache message.
	//About the response success/failed, the corresponding resource module
	// will do these things.	   
	tag := msg.GetTag()
	if tag == "" {
		// this is a message to twin.
		cm.context.Send(common.TwinModuleName, msg)
	} else {
		_ , exist := cm.context.MessageCache.Load(tag)
		if exist {
			cm.context.MessageCache.Delete(tag) 
		}
	}	
}

//dealMessageTimeout
func (cm *CommModule) dealMessageTimeout() {
	cm.context.MessageCache.Range(func (key interface{}, value interface{}) bool {
		msg, isMsgType := value.(*model.Message)
		if isMsgType {
			target := msg.GetTarget()
			operation := msg.GetOperation()

			timeStamp := msg.GetTimestamp()/1e3
			now	:= time.Now().UnixNano() / 1e9
			if now - timeStamp >= types.DGTWINS_MSG_TIMEOUT {
				if strings.Contains(target, common.DeviceName) {
					if strings.Compare(common.DGTWINS_OPS_RESPONSE, operation) != 0 {
						//mark device status is offline.
						//send package and tell twin module, device is offline.
						twinID := common.GetTwinID(msg)
						dgtwin := &common.DeviceTwin{
							ID: twinID,
							State:	common.DGTWINS_STATE_OFFLINE,
						}
		
						msgContent, err := common.BuildDeviceMessage(dgtwin)
						if err == nil {
							modelMsg := common.BuildModelMessage(types.MODULE_NAME, types.MODULE_NAME, common.DGTWINS_OPS_UPDATE, 
																	common.DGTWINS_RESOURCE_TWINS, msgContent)
							cm.context.SendToModule(types.DGTWINS_MODULE_TWINS, modelMsg)
						}
					}	
				}
				cm.context.MessageCache.Delete(key)
				return true
			}else{
				if strings.Contains(target, common.DeviceName) && 
						strings.Compare(common.DGTWINS_OPS_DETECT, operation) == 0 {
					// this is a ping message to device, then, we delete this mark
					// and make this state as offline.
					twinID := common.GetTwinID(msg)
					twinState := cm.context.GetTwinState(twinID)
					if twinState != common.DGTWINS_STATE_CREATED &&
					   		twinState != common.DGTWINS_STATE_OFFLINE {

						dgtwin := &common.DeviceTwin{
							ID: twinID,
							State:	common.DGTWINS_STATE_OFFLINE,
						}
		
						msgContent, err := common.BuildDeviceMessage(dgtwin)
						if err == nil {
							modelMsg := common.BuildModelMessage(types.MODULE_NAME, types.MODULE_NAME, common.DGTWINS_OPS_UPDATE, common.DGTWINS_RESOURCE_TWINS, msgContent)
							cm.context.SendToModule(types.DGTWINS_MODULE_TWINS, modelMsg)
						}
						
						klog.Infof("### Detect Device(%s) is offline", twinID)
					}
					cm.context.MessageCache.Delete(key)
				}else {
					//resend this message.
					klog.Infof("### Resend this message...")
					cm.context.SendToModule(types.DGTWINS_MODULE_COMM, msg)
				}
				return true
			}
		}

		return false
	})
}
