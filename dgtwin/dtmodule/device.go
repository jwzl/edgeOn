package dtmodule

import (
	"sync"
	"time"
	"strconv"
	"errors"
	"strings"
	"k8s.io/klog"
	"encoding/json"
	"github.com/jwzl/edgeOn/common"
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
	dm.deviceCommandTbl[common.DGTWINS_OPS_CREATE] = dm.twinsCreateHandle
	dm.deviceCommandTbl[common.DGTWINS_OPS_UPDATE] = dm.deviceUpdateHandle
	dm.deviceCommandTbl[common.DGTWINS_OPS_DELETE] = dm.deviceDeleteHandle	
	dm.deviceCommandTbl[common.DGTWINS_OPS_GET] = dm.deviceGetHandle	
	dm.deviceCommandTbl[common.DGTWINS_OPS_RESPONSE] = dm.deviceResponseHandle	
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

// twinsCreateHandle
// create twins is just only in cloud sides or edge/app, and
// device sides can't create twins.
func (dm *TwinModule) twinsCreateHandle(msg *model.Message) (interface{}, error) {
	var twinMsg	common.TwinMessage

	msgSource := msg.GetSource()
	// if from device, ignore this. 
	if strings.Contains(msgSource, common.DGTWINS_RESOURCE_DEVICE) {
		return nil, nil
	}

	content, ok := msg.Content.([]byte)
	if !ok {
		return nil, errors.New("invaliad message content")
	}
	klog.Infof("create twin message (%s)", string(content))
	err := json.Unmarshal(content, &twinMsg)
	if err != nil {
		return nil, err
	}
	
	twins := make([]common.DeviceTwin, 0)
	//get all requested twins
	for key, _ := range twinMsg.Twins	{
		twin := &twinMsg.Twins[key]
		//for each dgtwin
		twinID := twin.ID
		exist := dm.context.DGTwinIsExist(twinID)
		if !exist {
			dgTwin := &common.DigitalTwin{
				ID:	twinID,
				State: common.DGTWINS_STATE_OFFLINE,
			}
			
			//Create DGTwin is always success since it just create data startuctre
			// in memory  and database.
			//Infutre, we will store DGTwin into sqlite database. 
			dm.context.DGTwinList.Store(twinID, dgTwin)
			var deviceMutex	sync.Mutex
			dm.context.DGTwinMutex.Store(twinID, &deviceMutex)
			//save to sqlite, implement in future.
			//TODO:

			twins = append(twins, common.DeviceTwin{ID: twinID})	

			//detect the physical device	
			// send broadcast to all device, and wait (own this ID) device's response,
			// if it has reply, then will report all property of this device.
			content, _ = common.BuildDeviceMessage(twin)
			deviceMsg := common.BuildModelMessage(types.MODULE_NAME, "device-"+twinID, 
					common.DGTWINS_OPS_DETECT, common.DGTWINS_RESOURCE_DEVICE, content)
			klog.Infof("Send to device with (%v)", deviceMsg)
			dm.context.SendToModule(types.DGTWINS_MODULE_COMM, deviceMsg)
		}
	}
	
	//Send response.
	msgContent, err := common.BuildResponseMessage(common.RequestSuccessCode, "Success", twins)
	if err != nil {
		//Internal err			
		return nil,  err
	}else{
		dm.context.SendResponseMessage(msg, msgContent)
	}

	return nil, nil	
}

// handle device update.
// the message is just from device sides. cloud & edge/app can't update these information
// by this api.
func (dm *TwinModule) deviceUpdateHandle(msg *model.Message ) (interface{}, error) {
	var devMsg	common.DeviceMessage
	msgSource := msg.GetSource()

	// if from device, ignore this. 
	if strings.Contains(msgSource, common.DGTWINS_RESOURCE_DEVICE) != true &&
		strings.Contains(msgSource, common.TwinModuleName) != true {
		return nil, nil
	}

	content, ok := msg.Content.([]byte)
	if !ok {
		return nil, errors.New("invaliad message content")
	}
	klog.Infof("update twin message (%s)", string(content))
	err := json.Unmarshal(content, &devMsg)
	if err != nil {
		return nil, err
	}

	twinID := devMsg.Twin.ID
	exist := dm.context.DGTwinIsExist(twinID)
	if exist {
		//Update Twin
		dm.context.Lock(twinID)
		v, _ := dm.context.DGTwinList.Load(twinID)
		oldTwin, _ :=v.(*common.DigitalTwin)

		//deal device update
		err = dm.dealTwinUpdate(oldTwin, &devMsg.Twin)
		dm.context.Unlock(twinID)

		if err == nil {
			klog.Infof("######### (%s) is online  ##########", twinID)
			klog.Infof("######### Device information update successful  ##########")

			//if the update is from device directly, then reply it.
			if strings.Contains(msgSource, common.DGTWINS_RESOURCE_DEVICE) {
		
				content, _ = common.BuildDeviceResponseMessage(strconv.Itoa(common.RequestSuccessCode), 
										"update success", &common.DeviceTwin{ID: twinID})
				modelMsg := common.BuildModelMessage(types.MODULE_NAME, common.DeviceName, 
					common.DGTWINS_OPS_RESPONSE, common.DGTWINS_RESOURCE_DEVICE, content)

				klog.Infof("Device to device with (%v)", modelMsg)
				dm.context.SendToModule(types.DGTWINS_MODULE_COMM, modelMsg)
			}
			//notify others about device is online
		} else {
			//Internel err!
		}
	}else {
		//Ignore when twin is not exist.
	}
	
	return nil, nil
}

//deal twin update.
//this is a patch for the old device state.
func (dm *TwinModule) dealTwinUpdate(oldTwin *common.DigitalTwin, newTwin *common.DeviceTwin) error {
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
	if oldTwin.MetaData == nil {
		oldTwin.MetaData = make(map[string]*common.MetaType)
	} 
	if len(newTwin.MetaData) > 0 {
		for _ , meta := range newTwin.MetaData {
			oldTwin.MetaData[meta.Name] = &meta
		}
	}

	//update desired
	if len(newTwin.Properties.Desired) > 0 {
		if oldTwin.Properties.Desired == nil {
			oldTwin.Properties.Desired = make(map[string]*common.TwinProperty)
		}

		for _ , prop := range newTwin.Properties.Desired {
			oldTwin.Properties.Desired[prop.Name] = &prop
		}
	}	

	//update reported
	if len(newTwin.Properties.Reported) > 0 {
		if oldTwin.Properties.Reported == nil {
			oldTwin.Properties.Reported = make(map[string]*common.TwinProperty)
		}

		for _ , prop := range newTwin.Properties.Reported {
			oldTwin.Properties.Desired[prop.Name] = &prop
		}
	}	

	return nil
}

/*
* We just support delete a twin (device).
*/
func (dm *TwinModule) deviceDeleteHandle(msg *model.Message) (interface{}, error) {
	var twinMsg	common.TwinMessage

	msgSource := msg.GetSource()
	// if from device, ignore this. 
	if strings.Contains(msgSource, common.DGTWINS_RESOURCE_DEVICE) {
		return nil, nil
	}

	content, ok := msg.Content.([]byte)
	if !ok {
		return nil, errors.New("invaliad message content")
	}
	klog.Infof("delete twin message (%s)", string(content))
	err := json.Unmarshal(content, &twinMsg)
	if err != nil {
		return nil, err
	}

	if len(twinMsg.Twins) > 0 {
		var msgContent  []byte
		//get the first twin since we just support delete a twin.
		dgTwin := &twinMsg.Twins[0]

		twinID := dgTwin.ID
		exist := dm.context.DGTwinIsExist(twinID)
		if !exist {
			msgContent, err = common.BuildResponseMessage(common.NotFoundCode, "Not found", twinMsg.Twins)
			if err != nil {
				//Internal err.
				return nil, err
			}
		} else {
			//delete the device & mutex.
			dm.context.Lock(twinID)
			dm.context.DGTwinList.Delete(twinID)
			dm.context.Unlock(twinID)
			dm.context.DGTwinMutex.Delete(twinID)

			msgContent, err = common.BuildResponseMessage(common.RequestSuccessCode, "Deleted", twinMsg.Twins)
			if err != nil {
				//Internal err.
				return nil, err
			}
		}

		//notify the device delete link with dgtwin.
		dm.context.SendTwinMessage2Device(msg, common.DGTWINS_OPS_DELETE, twinMsg.Twins)

		dm.context.SendResponseMessage(msg, msgContent)
	}

	return nil, nil
}

//deviceGetHandle
// this function will return exist twin json profile to requester. 
// If request twin is not exit, this func will return empty list.
func (dm *TwinModule) deviceGetHandle(msg *model.Message) (interface{}, error) {
	var twinMsg	common.TwinMessage
	twins := make([]common.DeviceTwin, 0)

	content, ok := msg.Content.([]byte)
	if !ok {
		return nil, errors.New("invaliad message content")
	}

	err := json.Unmarshal(content, &twinMsg)
	if err != nil {
		return nil, err
	}

	for _, twin := range twinMsg.Twins	{
		//for each dgtwin
		twinID := twin.ID

		exist := dm.context.DGTwinIsExist(twinID)
		if exist {
			v, _ := dm.context.DGTwinList.Load(twinID)
			savedTwin, isDgTwinType  :=v.(*common.DigitalTwin)
			if !isDgTwinType {
				return nil,  errors.New("invalud digital twin type")
			}

			//convert digital twin to device twin. 
			deviceTwin := dm.Digital2Device(savedTwin)

			twins = append(twins, *deviceTwin)
		}else {
			// not exist, ignore.
		}
	}

	//Send the response.
	msgContent, err := common.BuildResponseMessage(common.RequestSuccessCode, "Get", twins)
	if err != nil {
		//Internal err.
		return nil, err
	}
	dm.context.SendResponseMessage(msg, msgContent)

	return nil, nil
}	

// deviceResponseHandle: handle response.
func (dm *TwinModule) deviceResponseHandle(msg *model.Message) (interface{}, error) {
	msgSource := msg.GetSource()

	// if from device. 
	if strings.Contains(msgSource, common.DGTWINS_RESOURCE_DEVICE) {
		//unMarshal device response message.
		resp, err := common.UnMarshalDeviceResponseMessage(msg)
		if err != nil {
    		return nil, err
   		}

		code, err := strconv.Atoi(resp.Code)
		if err != nil {
    		return nil, err
    	}

		switch code {
		case common.DeviceFound:
			//Mark the state is online.
			resp.Twin.State = common.DGTWINS_STATE_ONLINE

			content, _ := common.BuildDeviceMessage(&resp.Twin)
			deviceMsg := common.BuildModelMessage(types.MODULE_NAME, types.MODULE_NAME, 
					common.DGTWINS_OPS_UPDATE, common.DGTWINS_RESOURCE_TWINS, content)

			klog.Infof("Device is online, update device with (%v)", deviceMsg)
			dm.context.SendToModule(types.DGTWINS_MODULE_COMM, deviceMsg)
		case common.DeviceNotReady:
		}

	}else{
		// process the reply from cloud / edge/app.

	}

	// pass the message to common module to unmark the request message.
	// or, the request message will repeat send to physical device. 
	dm.context.SendToModule(types.DGTWINS_MODULE_COMM, msg)

	return nil, nil
}	


//PingDevice: ping device.
//device should reply the online code for that device is alive.  
func (dm *TwinModule) PingDevice() {
	dm.context.DGTwinList.Range(func(key, value interface{}) bool {
		twinID := key.(string)
		
		msgContent, err := common.BuildDeviceMessage(&common.DeviceTwin{ID: twinID})
		if err == nil {
			modelMsg := dm.context.BuildModelMessage(types.MODULE_NAME, common.DeviceName, 
								common.DGTWINS_OPS_SYNC, common.DGTWINS_RESOURCE_DEVICE, msgContent)
			dm.context.SendToModule(types.DGTWINS_MODULE_COMM, modelMsg)
		}

		return true	
	})	
}

// convert digital twins to device twins. 
func (dm *TwinModule) Digital2Device(savedTwin *common.DigitalTwin) *common.DeviceTwin {
	deviceTwin := &common.DeviceTwin{
		ID:			savedTwin.ID,
		Name:		savedTwin.Name,
		Description: savedTwin.Description,
		State:		savedTwin.State,
		LastState:  savedTwin.LastState,
		MetaData:   make([]common.MetaType, 0),
	}

	// get the metadata.
	for _ , meta := range savedTwin.MetaData {
		if meta != nil {
			deviceTwin.MetaData = append(deviceTwin.MetaData, *meta)
		}
	}

	//update desired
	if len(savedTwin.Properties.Desired) > 0 {
		if deviceTwin.Properties.Desired == nil {
			deviceTwin.Properties.Desired = make([]common.TwinProperty, 0)
		}

		for _ , prop := range savedTwin.Properties.Desired {
			deviceTwin.Properties.Desired = append(deviceTwin.Properties.Desired, *prop)
		}
	}

	//update Reported
	if len(savedTwin.Properties.Reported) > 0 {
		if deviceTwin.Properties.Reported == nil {
			deviceTwin.Properties.Reported = make([]common.TwinProperty, 0)
		}

		for _ , prop := range savedTwin.Properties.Reported {
			deviceTwin.Properties.Reported = append(deviceTwin.Properties.Reported, *prop)
		}
	}

	return deviceTwin
}
