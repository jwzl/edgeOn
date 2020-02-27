package dtmodule

import (
	"errors"
	"strings"
	"strconv"
	"k8s.io/klog"
	"encoding/json"
	"github.com/jwzl/edgeOn/common"
	"github.com/jwzl/wssocket/model"
	"github.com/jwzl/edgeOn/dgtwin/types"
	"github.com/jwzl/edgeOn/dgtwin/dtcontext"
)

type PropertyCmdFunc  func(msg *model.Message ) error
type PropActionHandle func(msg *model.Message, savedTwin *common.DigitalTwin,  msgTwin *common.DeviceTwin) error	
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

	pm.propertyCmdTbl[common.DGTWINS_OPS_UPDATE] = pm.propUpdateHandle
	pm.propertyCmdTbl[common.DGTWINS_OPS_DELETE] = pm.propDeleteHandle
	pm.propertyCmdTbl[common.DGTWINS_OPS_GET] = pm.propGetHandle
	pm.propertyCmdTbl[common.DGTWINS_OPS_WATCH] = pm.propWatchHandle
	pm.propertyCmdTbl[common.DGTWINS_OPS_SYNC] = pm.propSyncHandle
	pm.propertyCmdTbl[common.DGTWINS_OPS_RESPONSE] = pm.propResponseHandle
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
				klog.Infof("property message arrived {Header:%v Router:%v-}", 
												message.Header, message.Router)
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

//propUpdateHandle: handle update property. 
func (pm *PropertyModule) propUpdateHandle(msg *model.Message ) error {
	return pm.handleMessage(msg, func(msg *model.Message, savedTwin *common.DigitalTwin,  msgTwin *common.DeviceTwin) error{
		//savedTwin and msgTwin are always != nil
		twinID := savedTwin.ID 
		pm.context.Lock(twinID)
		
		if savedTwin.Properties.Desired == nil {
			savedTwin.Properties.Desired = make(map[string]*common.TwinProperty)
		}

		//if msgTwin.Properties != nil {
		savedDesired  := savedTwin.Properties.Desired	
		newDesired := msgTwin.Properties.Desired
			
		//Update twin property.
		for _ , prop := range newDesired {
			savedDesired[prop.Name] = &prop
		}
		//}	
		pm.context.Unlock(twinID)

		//Send the response
		msgRespWhere := msg.GetSource()
		twins := []common.DeviceTwin{*msgTwin}

		if strings.Compare(msgRespWhere, types.MODULE_NAME) != 0 {
			msgContent, err := common.BuildResponseMessage(common.RequestSuccessCode, "Success", twins)
			if err != nil {
				return err
			}else{
				//send the msg to comm module and process it
				pm.context.SendResponseMessage(msg, msgContent)
			}
		}

		// notify the device.
		pm.context.SendMessage2Device(common.DGTWINS_OPS_UPDATE, msgTwin)

		return nil
	})
}

//propDeleteHandle: delete property
func (pm *PropertyModule) propDeleteHandle(msg *model.Message ) error {
	
	return pm.handleMessage(msg, func(msg *model.Message, savedTwin *common.DigitalTwin, msgTwin *common.DeviceTwin) error{
		twinID := savedTwin.ID 
		pm.context.Lock(twinID)
		
		savedDesired  := savedTwin.Properties.Desired
		savedReported := savedTwin.Properties.Reported		
		newDesired := msgTwin.Properties.Desired
		newReported := msgTwin.Properties.Reported
		
		if newDesired != nil && len(newDesired) > 0 {
			if savedDesired == nil {
				pm.context.Unlock(twinID)
				pm.sendNotFoundPropMessage(msg, msgTwin)
				return nil
			}

			for _ , prop := range newDesired {
				if _, exist := savedDesired[prop.Name]; !exist {
					pm.context.Unlock(twinID)
					pm.sendNotFoundPropMessage(msg, msgTwin)
					return nil
				}
				delete(savedDesired, prop.Name)
			}
		}
		
		if newReported != nil && len(newReported) > 0 {
			if savedReported == nil {
				pm.context.Unlock(twinID)
				pm.sendNotFoundPropMessage(msg, msgTwin)
				return nil
			}

			for _ , prop := range newReported {
				if _, exist := savedReported[prop.Name]; !exist {
					pm.context.Unlock(twinID)
					pm.sendNotFoundPropMessage(msg, msgTwin)
					return nil
				}
				delete(savedReported, prop.Name)
			}
		}
		pm.context.Unlock(twinID)

		twins := []common.DeviceTwin{*msgTwin}
		msgContent, err := common.BuildResponseMessage(common.RequestSuccessCode, "Success", twins)
		if err != nil {
			return err
		}
			
		//send the msg to comm module and process it
		pm.context.SendResponseMessage(msg, msgContent)
		
		//send delete to device.  
		if (newReported != nil && len(newReported) > 0 ) || 
					(newDesired != nil && len(newDesired) > 0 ) {
			pm.context.SendMessage2Device(common.DGTWINS_OPS_DELETE, msgTwin)
		}
		
		return nil
	})
}

//propGetHandle: Get property.
func (pm *PropertyModule) propGetHandle (msg *model.Message ) error {
	return pm.handleMessage(msg, func(msg *model.Message, savedTwin *common.DigitalTwin, msgTwin *common.DeviceTwin) error{
		twinID := savedTwin.ID 
		
		pm.context.Lock(twinID)
		savedDesired  := savedTwin.Properties.Desired
		savedReported := savedTwin.Properties.Reported		
		newDesired := msgTwin.Properties.Desired
		newReported := msgTwin.Properties.Reported

		desiredProps := make([]common.TwinProperty, 0)
		if newDesired != nil && len(newDesired) > 0 {
			if savedDesired == nil {
				pm.context.Unlock(twinID)
				pm.sendNotFoundPropMessage(msg, msgTwin)
				return nil
			}

			for _, prop := range newDesired {
				if value, exist := savedDesired[prop.Name]; !exist {
					desiredProps = append(desiredProps, common.TwinProperty{Name: prop.Name})
					msgTwin.Properties.Desired = desiredProps
					msgTwin.Properties.Reported = nil
					msgTwin.State = savedTwin.State 
					
					pm.context.Unlock(twinID)

					pm.sendNotFoundPropMessage(msg, msgTwin)
					return nil
				}else {
					desiredProps = append(desiredProps, *value)
				}
			}
		}
		
		reportedProps := make([]common.TwinProperty, 0)
		if newReported != nil && len(newReported) > 0 {
			if savedReported == nil {
				pm.context.Unlock(twinID)
				pm.sendNotFoundPropMessage(msg, msgTwin)
				return nil
			}

			for _, prop := range newReported {
				if value, exist := savedReported[prop.Name]; !exist {
					reportedProps = append(reportedProps, common.TwinProperty{Name: prop.Name})
					msgTwin.Properties.Reported = reportedProps
					msgTwin.Properties.Desired = nil
					msgTwin.State = savedTwin.State 
					
					pm.context.Unlock(twinID)
	
					//Not found this message.
					pm.sendNotFoundPropMessage(msg, msgTwin)
					
					return nil
				}else {
					reportedProps = append(reportedProps, *value)
				}	
			}
		}

			
		msgTwin.Properties.Desired = desiredProps
		msgTwin.Properties.Reported = reportedProps
		msgTwin.State = savedTwin.State

		pm.context.Unlock(twinID)

		twins := []common.DeviceTwin{*msgTwin}
		msgContent, err := common.BuildResponseMessage(common.RequestSuccessCode, "Success", twins)
		if err != nil {
			return err
		}
		pm.context.SendResponseMessage(msg, msgContent)			

		return nil
	})
}

// propWatchHandle: handle property watch.
// We don't know we send the property's update to who , since there are 2 reciever(cloud and edge/app)
// So reciever must call watch to recieve these twin properties 's update.
// If Properties is nil or no  properties in request message, we consider it to watch all properties of 
// this twin.
func (pm *PropertyModule) propWatchHandle (msg *model.Message ) error {
	return pm.handleMessage(msg, func(msg *model.Message, savedTwin *common.DigitalTwin, msgTwin *common.DeviceTwin) error{
		/*twinID := savedTwin.ID 
		if savedTwin.Properties == nil || len(savedTwin.Properties.Reported) < 1 {
			pm.sendNotFoundPropMessage(msg, msgTwin)
		}else {
			watchEvent := types.CreateWatchEvent(msg.GetID(), twinID, msg.GetSource(), msg.GetResource())

			pm.context.Lock(twinID)

			savedReported := savedTwin.Properties.Reported	
			reportedProps := make(map[string]*types.PropertyValue)

			if msgTwin.Properties == nil || len(msgTwin.Properties.Reported) < 1 {
				reportedProps = savedReported
				for propName, _ := range savedReported {
					watchEvent.List = append(watchEvent.List, propName)
				}
			}else {	
				newReported := msgTwin.Properties.Reported
			
				for propName, _:= range newReported {
					if value, exist := savedReported[propName]; !exist {
						reportedProps[propName] = nil
						twinProperties:= &types.TwinProperties{
							Reported: reportedProps,
						}
						msgTwin.Properties = twinProperties
						msgTwin.State = savedTwin.State 
					
						pm.context.Unlock(twinID)
	
						pm.sendNotFoundPropMessage(msg, msgTwin)						

						return nil
					}else {
						watchEvent.List = append(watchEvent.List, propName)
						reportedProps[propName] = value
					}
				}		
			}

			twinProperties:= &types.TwinProperties{
				Reported: reportedProps,
			}
			msgTwin.Properties = twinProperties
			msgTwin.State = savedTwin.State

			pm.context.Unlock(twinID)
		
			twins := []*types.DigitalTwin{msgTwin}
			msgContent, err := types.BuildResponseMessage(types.RequestSuccessCode, "Success", twins)
			if err != nil {
				return err
			}
			pm.context.SendResponseMessage(msg, msgContent)

			//Cache the watch event
			pm.context.UpdateWatchCache(watchEvent)
		}*/

		return nil
	})
}

//propSyncHandle sync reported property from device.
// device will report the update of device's property automaticly.
func (pm *PropertyModule) propSyncHandle (msg *model.Message ) error {
	var twinMsg	common.DeviceMessage

	if msg.GetSource() != common.DeviceName {
		klog.Infof("we just process the SYNC data from device.")
		return nil
	}

	content, ok := msg.Content.([]byte)
	if !ok {
		return errors.New("invaliad message content")
	}

	err := json.Unmarshal(content, &twinMsg)
	if err != nil {
		return err
	}

	//Currently, we just support a twin's property's list since
	// we are just only foucus on the proprty's update.
	dgTwin := &twinMsg.Twin	
		
	twinID := dgTwin.ID
	
	exist := pm.context.DGTwinIsExist(twinID)
	if exist{
		v, _ := pm.context.DGTwinList.Load(twinID)
		savedTwin, isDgTwinType  :=v.(*common.DigitalTwin)
		if !isDgTwinType {
			return errors.New("invalud digital twin type")
		}
 		if savedTwin == nil {
			return errors.New("savedTwin=nil, Unexpected error")
		}	
		
		//1. save the data.
		pm.context.Lock(twinID)
		savedReported := savedTwin.Properties.Reported	
		newReported := dgTwin.Properties.Reported
			
		if newReported != nil && len(newReported) > 0 {
			if savedReported == nil {
				pm.context.Unlock(twinID)
				//ignore.
				return nil
			}

			for _ , prop := range newReported {
				if _, ok := savedReported[prop.Name]; ok {
					savedReported[prop.Name] = &prop
				}
			}
		}
		pm.context.Unlock(twinID)
		//2. send response to device .
		msgContent, err := common.BuildDeviceResponseMessage(strconv.Itoa(common.RequestSuccessCode), "SYNC Success", dgTwin)
		if err != nil {
			return err
		}
		pm.context.SendResponseMessage(msg, msgContent)
			
			//3. Check the cache event, if these property is cached, then send SYNC to target
			/*pm.context.RangeWatchCache(func(key, value interface{}) bool {
				ID := key.(string)
				if twinID != ID	{
					return true
				}
				
				pm.context.Lock(twinID)
				newReported := dgTwin.Properties.Reported
				syncProps := make(map[string]*types.PropertyValue)
				watchEvent := value.(*types.WatchEvent)
				for _, prop := range watchEvent.List {
					propValue, exist := newReported[prop]
					if exist {
						syncProps[prop] = propValue 
					}	
				}
				pm.context.Unlock(twinID)

				twinProperties:= &types.TwinProperties{
					Reported: syncProps,
				}
				dgTwin.Properties = twinProperties
				dgTwin.State = savedTwin.State

				twins := []*types.DigitalTwin{dgTwin}
				msgContent, err := types.BuildResponseMessage(types.RequestSuccessCode, "SYNC", twins)
				if err != nil {
					return false
				}
				pm.context.SendSyncMessage(watchEvent, msgContent)
				
				return true
			})*/
	}

	return nil
}

//handleMessage: General message process handle.
func (pm *PropertyModule) handleMessage (msg *model.Message, fn PropActionHandle) error {
	var twinMsg	common.TwinMessage

	content, ok := msg.Content.([]byte)
	if !ok {
		return errors.New("invaliad message content")
	}

	err := json.Unmarshal(content, &twinMsg)
	if err != nil {
		return err
	}

	//Currently, we just support a twin's property's list since
	// we are just only foucus on the proprty's update.
	if len(twinMsg.Twins) < 1 {
		klog.Warningf("invalid message format")
		//TODO:
	}

	for _, dgTwin := range twinMsg.Twins {
		twinID := dgTwin.ID
		exist := pm.context.DGTwinIsExist(twinID)
		if !exist {
			// Device has not created yet.
			twins := []common.DeviceTwin{dgTwin}
			msgContent, err := common.BuildResponseMessage(common.NotFoundCode, "Twin Not found", twins)
			if err != nil {
				return err
			}
			pm.context.SendResponseMessage(msg, msgContent)
		}else{
			v, _ := pm.context.DGTwinList.Load(twinID)
			savedTwin, isDgTwinType  :=v.(*common.DigitalTwin)
			if !isDgTwinType {
				return errors.New("invalud digital twin type")
			}
 			if savedTwin == nil {
				return errors.New("savedTwin=nil, Unexpected error")
			}	
			return fn(msg, savedTwin, &dgTwin)
		}
	}

	return nil
}

// propResponseHandle: handle all response.
func (pm *PropertyModule) propResponseHandle (msg *model.Message ) error {
	/*var resp types.DGTwinResponse

	content, ok := msg.Content.([]byte)
	if !ok {
		klog.Warningf("error message content format, ignore.")
		return errors.New("invaliad message content")
	}

	err := json.Unmarshal(content, &resp)
	if err != nil {
		return err
	}

	//TODO:
	// use can closed the watch.	

	if resp.Code != types.RequestSuccessCode {
		//TODO:
	}*/

	//Success
	pm.context.SendToModule(types.DGTWINS_MODULE_COMM, msg)

	return nil
}

func (pm *PropertyModule) sendNotFoundPropMessage(msg *model.Message, msgTwin *common.DeviceTwin) error {
	twins := []common.DeviceTwin{*msgTwin}
	msgContent, err := common.BuildResponseMessage(common.NotFoundCode, "twin No property/No this property", twins)
	if err != nil {
		return err
	}
			
	pm.context.SendResponseMessage(msg, msgContent)

	return nil
}
