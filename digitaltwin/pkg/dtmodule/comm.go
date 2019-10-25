package dtmodule

import (
	"strings"
	"k8s.io/klog"
	"github.com/jwzl/edgeOn/digitaltwin/pkg/dtcontext"
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

func NewCommModule(name string) *CommModule {
	return &CommModule{name:name}
}


func (cm *CommModule) Name() string {
	return cm.name
}


//Init the comm module.
func (cm *CommModule) Init_Module(dtc *dtcontext.DTContext, comm, heartBeat, confirm chan interface{}) {
	cm.context = dtc
	cm.recieveChan = comm
	cm.heartBeatChan = heartBeat
	cm.confirmChan = confirm
	//cm.initDeviceCommandTable()
}

//Start comm module
func (cm *CommModule) Start(){
	//Start loop.
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
				if strings.Compare("device", target) == 0 {
					//send to device.
					klog.Infof("send to device")	
					cm.sendMessageToDevice(message) 	
				}else if strings.HasPrefix(target, "cloud") {
					//send to message cloud.
					klog.Infof("send to cloud")
					cm.sendMessageToHub(message)	
				}else if strings.HasPrefix(target, "edge") {
					if strings.Compare(types.MODULE_NAME, target) == 0 {
						//this is response
						cm.dealMessageResponse(message)
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
		}
	}
}

// sendMessageToDevice
func (cm *CommModule) sendMessageToDevice(msg *model.Message) {
	operation := msg.GetOperation()

	if strings.Compare(types.DGTWINS_OPS_RESPONSE, operation) != 0 {
		//cache this message for confirm recieve the response.
		id := msg.GetID() 
		cm.context.MessageCache.Store(id, msg)
	}

	//send message to protocol bus.
	cm.context.Send("bus", msg)
}

//sendMessageToHub
func (cm *CommModule) sendMessageToHub(msg *model.Message) {
	//cache this message for confirm recieve the response.
	id := msg.GetID() 
	cm.context.MessageCache.Store(id, msg)

	//send message to message hub.
	cm.context.Send("hub", msg)
}

// dealMessageResponse
func (cm *CommModule) dealMessageResponse(msg *model.Message) {
	//If we recieve the response message, then delete cache message.
	//About the response success/failed, the corresponding resource module
	// will do these things.	   
	tag := msg.GetTag()
	v, exist := cm.context.MessageCache.Load(tag)
	if exist {
		cm.context.MessageCache.Delete(tag) 
	}	
}
