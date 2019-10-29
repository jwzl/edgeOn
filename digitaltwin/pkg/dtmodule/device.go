package dtmodule

import (
	"sync"
	"errors"
	"k8s.io/klog"
	"encoding/json"
	"github.com/jwzl/wssocket/model"
	"github.com/jwzl/edgeOn/digitaltwin/pkg/types"
	"github.com/jwzl/edgeOn/digitaltwin/pkg/dtcontext"
)

type DeviceCommandFunc  func(msg *model.Message )(interface{}, error)			
//this module process the device Create/delete/update/query.
type DeviceModule struct {
	// module name
	name			string
	context			*dtcontext.DTContext
	//for msg communication
	recieveChan		chan interface{}
	// for module's health check.
	heartBeatChan	chan interface{}
	confirmChan		chan interface{}
	deviceCommandTbl 	map[string]DeviceCommandFunc
}

func NewDeviceModule(name string) *DeviceModule {
	return &DeviceModule{name:name}
}

// Device command include: create/delete, update whole device, 
// Get whole device or device list.
func (dm *DeviceModule) initDeviceCommandTable() {
	dm.deviceCommandTbl = make(map[string]DeviceCommandFunc)
	dm.deviceCommandTbl["Update"] = dm.deviceUpdateHandle
	dm.deviceCommandTbl["Delete"] = dm.deviceDeleteHandle	
	dm.deviceCommandTbl["Get"] = dm.deviceGetHandle	
}

func (dm *DeviceModule) Name() string {
	return dm.name
}

//Init the device module.
func (dm *DeviceModule) InitModule(dtc *dtcontext.DTContext, comm, heartBeat, confirm chan interface{}) {
	dm.context = dtc
	dm.recieveChan = comm
	dm.heartBeatChan = heartBeat
	dm.confirmChan = confirm
	dm.initDeviceCommandTable()
}

//Start Device module
func (dm *DeviceModule) Start(){
	//Start loop.
	for {
		select {
		case msg, ok := <-dm.recieveChan:
			if !ok {
				//channel closed.
				return
			}
			
			message, isMsgType := msg.(*model.Message )
			if isMsgType {
		 		// do handle.
				if fn, exist := dm.deviceCommandTbl[message.GetOperation()]; exist {
					_, err := fn(message)
					if err != nil {
						klog.Errorf("Handle %s failed, ignored", message.Operation)
					}
				}else {
					klog.Errorf("No this handle for %s, ignored", message.Operation)
				}
			}
		case v, ok := <-dm.heartBeatChan:
			if !ok {
				return
			}
			
			err := dm.context.HandleHeartBeat(dm.Name(), v.(string))
			if err != nil {
				klog.Infof("%s module stopped", dm.Name())
				return
			}
		}
	}
}

// handle device create and update.
func (dm *DeviceModule)  deviceUpdateHandle(msg *model.Message ) (interface{}, error) {
	var dgTwinMsg types.DGTwinMessage 

	msgRespWhere := msg.GetSource()
	resource := msg.GetResource()

	content, ok := msg.Content.([]byte)
	if !ok {
		return nil, errors.New("invaliad message content")
	}

	err := json.Unmarshal(content, &dgTwinMsg)
	if err != nil {
		return nil, err
	}

	//get all requested twins
	for _, dgTwin := range dgTwinMsg.Twins	{
		//for each dgtwin
		deviceID := dgTwin.ID
		exist := dm.context.DGTwinIsExist(deviceID)
		if !exist {
			//Create DGTwin
			//Infutre, we will store DGTwin into sqlite database. 
			dm.context.DGTwinList.Store(deviceID, &dgTwin)
			var deviceMutex	sync.Mutex
			dm.context.DGTwinMutex.Store(deviceID, &deviceMutex)
			//save to sqlite, implement in future.
			//Send Response to target.	
			msgContent, err := types.BuildResponseMessage(types.RequestSuccessCode, "Success", nil)
			if err != nil {
				//Internal err.
				return nil,  err
			}else{
				modelMsg := dm.context.BuildModelMessage(types.MODULE_NAME, msgRespWhere, 
					types.DGTWINS_OPS_RESPONSE, resource, msgContent)
				//mark the request message id
				modelMsg.SetTag(message.GetID())	
				//send the msg to comm module and process it
				err := dm.context.SendToModule(types.DGTWINS_MODULE_COMM, modelMsg)
				if err != nil {
					//Internal error, Channel not found
					return nil, err
				}
			}	
			//notify device	
			// send broadcast to all device, and wait (own this ID) device's response,
			// if it has reply, then means that device is online.
			deviceMsg := dm.context.BuildModelMessage(types.MODULE_NAME, "device", 
						types.DGTWINS_OPS_DEVCREATE, types.DGTWINS_RESOURCE_DEVICE, content)

			err = dm.context.SendToModule(types.DGTWINS_MODULE_COMM, deviceMsg)
			if err != nil {
				//Internal error, Channel not found
				return nil, err
			}  			
		}else {
			//Update DGTwin
			dm.context.Lock(deviceID)
			v, exist := dm.context.DGTwinList.Load(deviceID)
			if !exist {
				return nil, errors.New("No such dgtwin in DGTwinList")
			}
			oldTwin, isDgTwinType  :=v.(*types.DigitalTwin)
			if !isDgTwinType {
				return nil,  errors.New("invalud digital twin type")
			}

			//deal device update
			dm.dealTwinUpdate(oldTwin, dgTwin)
			dm.context.Unlock(deviceID)

			//if message's source is not edge/dgtwin, send response.
			if strings.Compare(msgRespWhere, types.MODULE_NAME) != 0 {
				msgContent, err := types.BuildResponseMessage(types.RequestSuccessCode, "Success", nil)
				if err != nil {
					//Internal err.
					return nil,  err
				}else{
					modelMsg := dm.context.BuildModelMessage(types.MODULE_NAME, msgRespWhere, 
												types.DGTWINS_OPS_RESPONSE, resource, msgContent)
					//mark the request message id
					modelMsg.SetTag(msg.GetID())
					dm.context.SendToModule(types.DGTWINS_MODULE_COMM, modelMsg)
				}	
			}

			//if the twin has property, let property module to do it.
			if len(dgTwin.Properties) > 0 {
				twins := []types.DigitalTwin{dgTwin}
				bytes, err := types.BuildTwinMessage(types.DGTWINS_OPS_TWINSUPDATE, twins)
				modelMsg := dm.context.BuildModelMessage(types.MODULE_NAME, types.MODULE_NAME, 
										types.DGTWINS_OPS_TWINSUPDATE, types.DGTWINS_MODULE_PROPERTY, bytes)
				cm.context.SendToModule(types.DGTWINS_MODULE_PROPERTY, modelMsg)
			}
		}
	}
	
	return nil, nil
}

//deal twin update.
//this is a patch for the old device state.
func (dm *DeviceModule) dealTwinUpdate(oldTwin, newTwin *types.DigitalTwin) error {
	if oldTwin == nil || newTwin == nil {
		return errors.New("error oldTwin or newTwin")
	}

	klog.Infof("old twin =(%v), newTwin =(%v)", oldTwin, newTwin)
	if len(newTwin.Name) > 0 {
		oldTwin.Name = newTwin.Name
	}
	if len(newTwin.Description) > 0 {
		oldTwin.Description = newTwin.Description
	}
	if len(newTwin.State) > 0 {
		oldTwin.LastState = oldTwin.State 
		oldTwin.State = newTwin.State		
	}
	//patch all metadata to oldTwin. 
	if len(newTwin.MetaData) > 0 {
		for key, value := range newTwin.MetaData {
			oldTwin.MetaData[key] = value
		}
	}

	return nil
}

func (dm *DeviceModule)  deviceDeleteHandle(msg interface{}) (interface{}, error) {
	message, isMsgType := msg.(*model.Message)
	if !isMsgType {
		return nil, errors.New("invaliad message type")
	}

	return message, nil
}

func (dm *DeviceModule)  deviceGetHandle(msg interface{}) (interface{}, error) {
	message, isMsgType := msg.(*model.Message)
	if !isMsgType {
		return nil, errors.New("invaliad message type")
	}

	return message, nil
}		
