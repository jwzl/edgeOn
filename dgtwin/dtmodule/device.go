package dtmodule

import (
	"sync"
	"time"
	"errors"
	"strings"
	"k8s.io/klog"
	"encoding/json"
	"github.com/jwzl/wssocket/model"
	"github.com/jwzl/edgeOn/dgtwin/types"
	"github.com/jwzl/edgeOn/dgtwin/dtcontext"
)

type DeviceCommandFunc  func(msg *model.Message )(interface{}, error)			
//this module process the device Create/delete/update/query.
type TwinModule struct {
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

func NewTwinModule() *TwinModule {
	return &TwinModule{name: types.DGTWINS_MODULE_TWINS}
}

// Device command include: create/delete, update whole device, 
// Get whole device or device list.
func (dm *TwinModule) initDeviceCommandTable() {
	dm.deviceCommandTbl = make(map[string]DeviceCommandFunc)
	dm.deviceCommandTbl[types.DGTWINS_OPS_UPDATE] = dm.deviceUpdateHandle
	dm.deviceCommandTbl[types.DGTWINS_OPS_DELETE] = dm.deviceDeleteHandle	
	dm.deviceCommandTbl[types.DGTWINS_OPS_GET] = dm.deviceGetHandle	
	dm.deviceCommandTbl[types.DGTWINS_OPS_RESPONSE] = dm.deviceResponseHandle	
}

func (dm *TwinModule) Name() string {
	return dm.name
}

//Init the device module.
func (dm *TwinModule) InitModule(dtc *dtcontext.DTContext, comm, heartBeat, confirm chan interface{}) {
	dm.context = dtc
	dm.recieveChan = comm
	dm.heartBeatChan = heartBeat
	dm.confirmChan = confirm
	dm.initDeviceCommandTable()
}

//Start Device module
func (dm *TwinModule) Start(){
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
				klog.Infof("twin message arrived {Header:%v Router:%v-}", 
												message.Header, message.Router)
		 		// do handle.
				if fn, exist := dm.deviceCommandTbl[message.GetOperation()]; exist {
					_, err := fn(message)
					if err != nil {
						klog.Errorf("Handle %s failed, ignored", message.GetOperation())
					}
				}else {
					klog.Errorf("No this handle for %s, ignored", message.GetOperation())
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
		case <-time.After(120*time.Second):
			//Check & sync device's state.
			dm.PingDevice()	
		}
	}
}

// handle device create and update.
func (dm *TwinModule) deviceUpdateHandle(msg *model.Message ) (interface{}, error) {
	var dgTwinMsg types.DGTwinMessage 
	twins := make([]*types.DigitalTwin, 0)

	msgRespWhere := msg.GetSource()
	content, ok := msg.Content.([]byte)
	if !ok {
		return nil, errors.New("invaliad message content")
	}
	klog.Infof("update twin message (%s)", string(content))
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
			//Create DGTwin is always success since it just create data startuctre
			// in memory  and database.
			//Infutre, we will store DGTwin into sqlite database. 
			dm.context.DGTwinList.Store(deviceID, dgTwin)
			var deviceMutex	sync.Mutex
			dm.context.DGTwinMutex.Store(deviceID, &deviceMutex)
			//save to sqlite, implement in future.	

			if true { //always do true above.
				twins = append(twins, &types.DigitalTwin{ID:deviceID})
			}

			//notify device	
			// send broadcast to all device, and wait (own this ID) device's response,
			// if it has reply, then means that device is online.
			deviceMsg := dm.context.BuildModelMessage(types.MODULE_NAME, "device", 
					types.DGTWINS_OPS_CREATE, types.DGTWINS_RESOURCE_DEVICE, content)
			klog.Infof("Send to device with (%v)", deviceMsg)
			dm.context.SendToModule(types.DGTWINS_MODULE_COMM, deviceMsg)
		}else {
			//Update DGTwin
			dm.context.Lock(deviceID)
			v, _ := dm.context.DGTwinList.Load(deviceID)
			oldTwin, _ :=v.(*types.DigitalTwin)

			//deal device update
			err = dm.dealTwinUpdate(oldTwin, dgTwin)
			dm.context.Unlock(deviceID)

			if err == nil {
				twins = append(twins, &types.DigitalTwin{ID:deviceID})
			}
		}
	}

	//if message's source is not edge/dgtwin, send response.
	if strings.Compare(msgRespWhere, types.MODULE_NAME) != 0 {
		msgContent, err := types.BuildResponseMessage(types.RequestSuccessCode, "Success", twins)
		if err != nil {
			//Internal err			
			return nil,  err
		}else{
			dm.context.SendResponseMessage(msg, msgContent)
		}	
	}

	return nil, nil
}

//deal twin update.
//this is a patch for the old device state.
func (dm *TwinModule) dealTwinUpdate(oldTwin, newTwin *types.DigitalTwin) error {
	if oldTwin == nil || newTwin == nil {
		return errors.New("error oldTwin or newTwin")
	}

	oldJSON, _ := json.Marshal(oldTwin)	
	newJSON, _ := json.Marshal(newTwin)
	klog.Infof("oldJSON = %s, newJSON = %s", oldJSON, newJSON)		

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

	//if the twin no property, We just do create operation, and don't
	//change any value of the property.
	dgTwin := &types.DigitalTwin{
		ID: newTwin.ID,
		Properties: &types.TwinProperties{
			Desired:	make(map[string]*types.PropertyValue),
			Reported:	make(map[string]*types.PropertyValue),
		},
	}
	twins := []*types.DigitalTwin{dgTwin}

	//update desired
	if newTwin.Properties != nil && len(newTwin.Properties.Desired) > 0 {
		if oldTwin.Properties == nil {
			oldTwin.Properties = &types.TwinProperties{
				Desired:	make(map[string]*types.PropertyValue),
				Reported:	make(map[string]*types.PropertyValue),					
			}
		}
		if oldTwin.Properties.Desired == nil {
			oldTwin.Properties.Desired = make(map[string]*types.PropertyValue)
		}

		for prop, value := range newTwin.Properties.Desired {
			if _, ok := oldTwin.Properties.Desired[prop];!ok {
				dgTwin.Properties.Desired[prop] = value
			}
		} 
		
	}

	//update reported.
	if newTwin.Properties != nil && len(newTwin.Properties.Reported) > 0 {
		if oldTwin.Properties == nil {
			oldTwin.Properties = &types.TwinProperties{
				Desired:	make(map[string]*types.PropertyValue),
				Reported:	make(map[string]*types.PropertyValue),					
			}
		}
		if oldTwin.Properties.Reported == nil {
			oldTwin.Properties.Reported = make(map[string]*types.PropertyValue)
		}

		for prop, value := range newTwin.Properties.Reported {
			if _, ok := oldTwin.Properties.Reported[prop];!ok {
				dgTwin.Properties.Reported[prop] = value
			}
		} 
		
	}

	//Send update to property sub-module, and no response. 
	if len(dgTwin.Properties.Reported) > 0 || len(dgTwin.Properties.Desired) > 0	{
		bytes, err := types.BuildTwinMessage(types.DGTWINS_OPS_UPDATE, twins)
		if err == nil {
			modelMsg := dm.context.BuildModelMessage(types.MODULE_NAME, types.MODULE_NAME, 
								types.DGTWINS_OPS_UPDATE, types.DGTWINS_MODULE_PROPERTY, bytes)
			klog.Infof("modelMsg [%v]", modelMsg)
			dm.context.SendToModule(types.DGTWINS_MODULE_PROPERTY, modelMsg)
		}
	}

	return nil
}

/*
* We just support delete a twin (device).
*/
func (dm *TwinModule) deviceDeleteHandle(msg *model.Message) (interface{}, error) {
	var dgTwinMsg types.DGTwinMessage 

	content, ok := msg.Content.([]byte)
	if !ok {
		return nil, errors.New("invaliad message content")
	}

	err := json.Unmarshal(content, &dgTwinMsg)
	if err != nil {
		return nil, err
	}

	if len(dgTwinMsg.Twins) > 0	{
		var msgContent  []byte

		//get the first twin since we just support delete a twin.
		dgTwin := dgTwinMsg.Twins[0]		
		deviceID := dgTwin.ID
		twins := []*types.DigitalTwin{dgTwin}

		exist := dm.context.DGTwinIsExist(deviceID)
		if !exist {
			msgContent, err = types.BuildResponseMessage(types.NotFoundCode, "Not found", twins)
			if err != nil {
				//Internal err.
				return nil, err
			}
		}else {
			//delete the device & mutex.
			dm.context.Lock(deviceID)
			dm.context.DGTwinList.Delete(deviceID)
			dm.context.Unlock(deviceID)
			dm.context.DGTwinMutex.Delete(deviceID)

			msgContent, err = types.BuildResponseMessage(types.RequestSuccessCode, "Deleted", twins)
			if err != nil {
				//Internal err.
				return nil, err
			}
		}
		//notify the device delete link with dgtwin.
		dm.context.SendTwinMessage2Device(msg, types.DGTWINS_OPS_DELETE, twins)

		dm.context.SendResponseMessage(msg, msgContent)
	}

	return nil, nil
}

func (dm *TwinModule) deviceGetHandle(msg *model.Message) (interface{}, error) {
	var dgTwinMsg types.DGTwinMessage 
	twins := make([]*types.DigitalTwin, 0)

	content, ok := msg.Content.([]byte)
	if !ok {
		return nil, errors.New("invaliad message content")
	}

	err := json.Unmarshal(content, &dgTwinMsg)
	if err != nil {
		return nil, err
	}

	for _, dgTwin := range dgTwinMsg.Twins	{
		//for each dgtwin
		deviceID := dgTwin.ID

		exist := dm.context.DGTwinIsExist(deviceID)
		if exist {
			v, _ := dm.context.DGTwinList.Load(deviceID)
			savedTwin, isDgTwinType  :=v.(*types.DigitalTwin)
			if !isDgTwinType {
				return nil,  errors.New("invalud digital twin type")
			}
			twins = append(twins, savedTwin)
		}else {
			twins := []*types.DigitalTwin{dgTwin}
			msgContent, err := types.BuildResponseMessage(types.NotFoundCode, "Not found", twins)
			if err != nil {
				//Internal err.
				return nil, err
			}
			dm.context.SendResponseMessage(msg, msgContent)
			return nil, nil
		}
	}

	//Send the response.
	msgContent, err := types.BuildResponseMessage(types.RequestSuccessCode, "Get", twins)
	if err != nil {
		//Internal err.
		return nil, err
	}
	dm.context.SendResponseMessage(msg, msgContent)

	return nil, nil
}	

// deviceResponseHandle: handle response.
func (dm *TwinModule) deviceResponseHandle(msg *model.Message) (interface{}, error) {
	var resp types.DGTwinResponse

	content, ok := msg.Content.([]byte)
	if !ok {
		klog.Warningf("error message content format, ignore.")
		return nil, errors.New("invaliad message content")
	}

	err := json.Unmarshal(content, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Code == types.OnlineCode {
		if len(resp.Twins) > 0 {
			for _, dgTwin := range resp.Twins {
				if dgTwin == nil {
					klog.Infof("Twin is nil, Ignored")
					continue
				}

				twinID := dgTwin.ID
				v, _ := dm.context.DGTwinList.Load(twinID)
				savedTwin, isThisType:=v.(*types.DigitalTwin)
				if isThisType && savedTwin != nil {
					dm.context.Lock(twinID)
					state := savedTwin.State
					savedTwin.State = types.DGTWINS_STATE_ONLINE
					savedTwin.LastState = state	
					dm.context.Unlock(twinID)

					if state != types.DGTWINS_STATE_ONLINE {
						twins := []*types.DigitalTwin{savedTwin}
		
						msgContent, err := types.BuildTwinMessage(types.DGTWINS_OPS_UPDATE, twins)
						if err == nil {
							modelMsg := dm.context.BuildModelMessage(types.MODULE_NAME, "device", types.DGTWINS_OPS_UPDATE, 
																	"device", msgContent)
							dm.context.SendToModule(types.DGTWINS_MODULE_COMM, modelMsg)
						}
					}
				}			
			}
		}  
	}

	dm.context.SendToModule(types.DGTWINS_MODULE_COMM, msg)

	return nil, nil
}	


//PingDevice: ping device. 
func (dm *TwinModule) PingDevice() {
	dm.context.DGTwinList.Range(func(key, value interface{}) bool {
		twinID := key.(string)
		twin := &types.DigitalTwin{
			ID: twinID,	
		}	
		twins := []*types.DigitalTwin{twin}
		
		msgContent, err := types.BuildTwinMessage(types.DGTWINS_OPS_SYNC, twins)
		if err == nil {
			modelMsg := dm.context.BuildModelMessage(types.MODULE_NAME, "device", types.DGTWINS_OPS_SYNC, 
																	"device", msgContent)
			dm.context.SendToModule(types.DGTWINS_MODULE_COMM, modelMsg)
		}

		return true	
	})	
}
