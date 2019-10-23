package dtmodule

import (
	"k8s.io/klog"
	"github.com/jwzl/edgeOn/digitaltwin/pkg/dtcontext"
)

type DeviceCommandFunc  func(msg interface{})(interface{}, error)			
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
	dm.deviceCommandTbl["Update"] = deviceUpdateHandle
	dm.deviceCommandTbl["Delete"] = deviceDeleteHandle	
	dm.deviceCommandTbl["Get"] = deviceGetHandle	
}

func (dm *DeviceModule) Name() string {
	return dm.name
}

//Init the device module.
func (dm *DeviceModule) Init_Module(dtc *dtcontext.DTContext, comm, heartBeat, confirm chan interface{}) {
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
			
			message, isDTMsg := msg.(*types.DTMessage)
			if isDTMsg {
		 		// do handle.
				if fn, exist := dm.deviceCommandTbl[message.Operation]; exist {
					_, err := fn(message.Msg)
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
func (dm *DeviceModule)  deviceUpdateHandle(msg interface{}) (interface{}, error) {
	var dgTwin types.DigitalTwin
	message, isMsgType := msg.(*model.Message)
	if !isMsgType {
		return nil, errors.New("invaliad message type")
	}
	content, ok := message.Content.([]byte)
	if !ok {
		return nil, errors.New("invaliad message content")
	}

	err := json.Unmarshal(content, &dgTwin)
	if err != nil {
		return nil, err
	}
	
	deviceID := dgTwin.ID
	exist := dm.context.DGTwinIsExist(deviceID)
	if !exist {
		//Create DGTwin
				
	}else {
		//Update DGTwin
	}
	// Read from sqlite

	//save to sqlite

	//send the resonpose.
}

func (dm *DeviceModule)  deviceDeleteHandle(msg interface{}) (interface{}, error) {
	message, isMsgType := msg.(*model.Message)
	if !isMsgType {
		return nil, errors.New("invaliad message type")
	}
}

func (dm *DeviceModule)  deviceGetHandle(msg interface{}) (interface{}, error) {
	message, isMsgType := msg.(*model.Message)
	if !isMsgType {
		return nil, errors.New("invaliad message type")
	}
}		
