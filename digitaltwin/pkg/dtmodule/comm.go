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
				}else if strings.HasPrefix(target, "cloud") {
					//
				}else if strings.HasPrefix(target, "edge") {


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
