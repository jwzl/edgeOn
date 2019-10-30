package dtmodule

import (
	"sync"
	"errors"
	"strings"
	"k8s.io/klog"
	"encoding/json"
	"github.com/jwzl/wssocket/model"
	"github.com/jwzl/edgeOn/digitaltwin/pkg/types"
	"github.com/jwzl/edgeOn/digitaltwin/pkg/dtcontext"
)

type PropertyCmdFunc  func(msg *model.Message ) error
type PropertyModule struct {
	// module name
	name			string
	context			*dtcontext.DTContext
	//for msg communication
	recieveChan		chan interface{}
	// for module's health check.
	heartBeatChan	chan interface{}
	confirmChan		chan interface{}
	propertyCmdTbl 	map[string]PropertyCmdFunc
}	

func NewPropertyModule() *PropertyModule {
	return &PropertyModule{name: types.DGTWINS_MODULE_PROPERTY}
}

func (pm *PropertyModule) Name() string {
	return pm.name
}

func (pm *PropertyModule) initPropertyCmdTbl() {
	pm.propertyCmdTbl = make(map[string]PropertyCmdFunc)

	pm.propertyCmdTbl[types.DGTWINS_OPS_UPDATE] = pm.propUpdateHandle
	pm.propertyCmdTbl[types.DGTWINS_OPS_DELETE] = pm.propDeleteHandle
	pm.propertyCmdTbl[types.DGTWINS_OPS_GET] = pm.propGetHandle
	pm.propertyCmdTbl[types.DGTWINS_OPS_WATCH] = pm.propWatchHandle
	pm.propertyCmdTbl[types.DGTWINS_OPS_SYNC] = pm.propSyncHandle
}

func (pm *PropertyModule) InitModule(dtc *dtcontext.DTContext, comm, heartBeat, confirm chan interface{}) {
	pm.context = dtc
	pm.recieveChan = comm
	pm.heartBeatChan = heartBeat
	pm.confirmChan = confirm
	pm.initPropertyCmdTbl()
}

func (pm *PropertyModule) Start() {
	//Start loop.
	for {
		select {
		case msg, ok := <-pm.recieveChan:
			if !ok {
				//channel closed.
				return
			}
			
			message, isMsgType := msg.(*model.Message)
			if isMsgType {
				if fn, exist := pm.propertyCmdTbl[message.GetOperation()]; exist {
					err := fn(message)
					if err != nil {
						klog.Errorf("Handle failed, ignored (%v)", message)
					}
				}else {
					klog.Errorf("No this handle for %s, ignored", message.GetOperation())
				}
			}
		case v, ok := <-pm.heartBeatChan:
			if !ok {
				return
			}
			
			err := pm.context.HandleHeartBeat(pm.Name(), v.(string))
			if err != nil {
				klog.Infof("%s module stopped", pm.Name())
				return
			}
		}
	}
}

func (pm *PropertyModule) propUpdateHandle(msg *model.Message ) error {
	var dgTwinMsg types.DGTwinMessage 

	msgRespWhere := msg.GetSource()
	resource := msg.GetResource()

	content, ok := msg.Content.([]byte)
	if !ok {
		return errors.New("invaliad message content")
	}

	err := json.Unmarshal(content, &dgTwinMsg)
	if err != nil {
		return err
	}

	//Currently, we just support update a twin's property's list since
	// we are just only foucus on the proprty's update.
	if len(dgTwinMsg.Twins) < 1 {
		klog.Warningf("invalid message format")
		//TODO:
	}
	for _, dgTwin := range dgTwinMsg.Twins	{
		deviceID := dgTwin.ID
		exist := dm.context.DGTwinIsExist(deviceID)
		if !exist {
		
		}else{

		}
	}
}
func (pm *PropertyModule) propDeleteHandle(msg *model.Message ) error {

}

func (pm *PropertyModule) propGetHandle (msg *model.Message ) error {

}

func (pm *PropertyModule) propWatchHandle (msg *model.Message ) error {

}

func (pm *PropertyModule) propSyncHandle (msg *model.Message ) error {

}
