package dtmodule

import (
	_"sync"
	"errors"
	_"strings"
	"k8s.io/klog"
	"encoding/json"
	"github.com/jwzl/wssocket/model"
	"github.com/jwzl/edgeOn/digitaltwin/pkg/types"
	"github.com/jwzl/edgeOn/digitaltwin/pkg/dtcontext"
)

type PropertyCmdFunc  func(msg *model.Message ) error
type PropActionHandle func(msg *model.Message, savedTwin, msgTwin *types.DigitalTwin) error	
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

//propUpdateHandle: handle create/update property. 
func (pm *PropertyModule) propUpdateHandle(msg *model.Message ) error {
	return pm.handleMessage(msg, func(msg *model.Message, savedTwin, msgTwin *types.DigitalTwin) error{
		//savedTwin and msgTwin are always != nil
		twinID := savedTwin.ID 
		pm.context.Lock(twinID)
		if savedTwin.Properties == nil {
			savedTwin.Properties = &types.TwinProperties{}
		}

		if msgTwin.Properties != nil {
			savedDesired  := savedTwin.Properties.Desired
			savedReported := savedTwin.Properties.Reported		
			newDesired := msgTwin.Properties.Desired
			newReported := msgTwin.Properties.Reported
	
			//Update twin property.
			for key, value := range newDesired {
				savedDesired[key] = value
			}
		
			for key, value := range newReported {
				savedReported[key] = value
			}
		}	
		pm.context.Unlock(twinID)

		//Send the response
		twins := []*types.DigitalTwin{msgTwin}
		msgContent, err := types.BuildResponseMessage(types.RequestSuccessCode, "Success", twins)
		if err != nil {
			return err
		}else{
			//send the msg to comm module and process it
			pm.context.SendResponseMessage(msg, msgContent)
		}

		// notify the device.
		pm.context.SendTwinMessage2Device(msg, types.DGTWINS_OPS_UPDATE, twins)
		return nil
	})
}

func (pm *PropertyModule) propDeleteHandle(msg *model.Message ) error {
	
	return pm.handleMessage(msg, func(msg *model.Message, savedTwin, msgTwin *types.DigitalTwin) error{

	
		return nil
	})
}

func (pm *PropertyModule) propGetHandle (msg *model.Message ) error {
	return pm.handleMessage(msg, func(msg *model.Message, savedTwin, msgTwin *types.DigitalTwin) error{


		return nil
	})
}

func (pm *PropertyModule) propWatchHandle (msg *model.Message ) error {
	return pm.handleMessage(msg, func(msg *model.Message, savedTwin, msgTwin *types.DigitalTwin) error{


		return nil
	})
}

func (pm *PropertyModule) propSyncHandle (msg *model.Message ) error {
	return pm.handleMessage(msg, func(msg *model.Message, savedTwin, msgTwin *types.DigitalTwin) error{


		return nil
	})
}

//handleMessage: General message process handle.
func (pm *PropertyModule) handleMessage (msg *model.Message, fn PropActionHandle) error {
	var dgTwinMsg types.DGTwinMessage 

	content, ok := msg.Content.([]byte)
	if !ok {
		return errors.New("invaliad message content")
	}

	err := json.Unmarshal(content, &dgTwinMsg)
	if err != nil {
		return err
	}

	//Currently, we just support a twin's property's list since
	// we are just only foucus on the proprty's update.
	if len(dgTwinMsg.Twins) < 1 {
		klog.Warningf("invalid message format")
		//TODO:
	}

	for _, dgTwin := range dgTwinMsg.Twins	{
		if dgTwin == nil {
			klog.Infof("Twin is nil, Ignored")
			continue
		}
		deviceID := dgTwin.ID
		exist := pm.context.DGTwinIsExist(deviceID)
		if !exist {
			// Device has not created yet.
			twins := []*types.DigitalTwin{dgTwin}
			msgContent, err := types.BuildResponseMessage(types.NotFoundCode, "Twin Not found", twins)
			if err != nil {
				return err
			}
			pm.context.SendResponseMessage(msg, msgContent)
		}else{
			v, _ := pm.context.DGTwinList.Load(deviceID)
			savedTwin, isDgTwinType  :=v.(*types.DigitalTwin)
			if !isDgTwinType {
				return errors.New("invalud digital twin type")
			}
 			if savedTwin == nil {
				return errors.New("savedTwin=nil, Unexpected error")
			}	
			return fn(msg, savedTwin, dgTwin)
		}
	}

	return nil
}
