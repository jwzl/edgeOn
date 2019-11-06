package dtmodule

import (
	"errors"
	"k8s.io/klog"
	"encoding/json"
	"github.com/jwzl/wssocket/model"
	"github.com/jwzl/edgeOn/dgtwin/pkg/types"
	"github.com/jwzl/edgeOn/dgtwin/pkg/dtcontext"
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
	pm.propertyCmdTbl[types.DGTWINS_OPS_RESPONSE] = pm.propResponseHandle
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
		if savedTwin.Properties.Desired == nil {
			savedTwin.Properties.Desired = make(map[string]*types.PropertyValue)
		}
		if savedTwin.Properties.Reported == nil {
			savedTwin.Properties.Reported = make(map[string]*types.PropertyValue)
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
			//TODO: infuture we will delete it, since reported can be set. 
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
		if msgTwin.Properties != nil {
			pm.context.SendTwinMessage2Device(msg, types.DGTWINS_OPS_UPDATE, twins)
		}
		return nil
	})
}

//propDeleteHandle: delete property
func (pm *PropertyModule) propDeleteHandle(msg *model.Message ) error {
	
	return pm.handleMessage(msg, func(msg *model.Message, savedTwin, msgTwin *types.DigitalTwin) error{
		twinID := savedTwin.ID 
		pm.context.Lock(twinID)
		if savedTwin.Properties == nil || msgTwin.Properties == nil {
			pm.context.Unlock(twinID)
			// send not found property message.
			pm.sendNotFoundPropMessage(msg, msgTwin)
		}else {
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
				for key, _ := range newDesired {
					if _, exist := savedDesired[key]; !exist {
						pm.context.Unlock(twinID)
						pm.sendNotFoundPropMessage(msg, msgTwin)
						return nil
					}
					delete(savedDesired, key)
				}
			}
		
			if newReported != nil && len(newReported) > 0 {
				if savedReported == nil {
					pm.context.Unlock(twinID)
					pm.sendNotFoundPropMessage(msg, msgTwin)
					return nil
				}

				for key, _ := range newReported {
					if _, exist := savedReported[key]; !exist {
						pm.context.Unlock(twinID)
						pm.sendNotFoundPropMessage(msg, msgTwin)
						return nil
					}
					delete(savedReported, key)
				}
			}
			pm.context.Unlock(twinID)

			twins := []*types.DigitalTwin{msgTwin}
			msgContent, err := types.BuildResponseMessage(types.RequestSuccessCode, "Success", twins)
			if err != nil {
				return err
			}
			
			//send the msg to comm module and process it
			pm.context.SendResponseMessage(msg, msgContent)
			//send delete to device.  
			if (newReported != nil && len(newReported) > 0 ) || 
					(newDesired != nil && len(newDesired) > 0 ) {
				pm.context.SendTwinMessage2Device(msg, types.DGTWINS_OPS_DELETE, twins)
			}
		}
		
		return nil
	})
}

//propGetHandle: Get property.
func (pm *PropertyModule) propGetHandle (msg *model.Message ) error {
	return pm.handleMessage(msg, func(msg *model.Message, savedTwin, msgTwin *types.DigitalTwin) error{
		twinID := savedTwin.ID 
		if savedTwin.Properties == nil || msgTwin.Properties == nil {

			pm.sendNotFoundPropMessage(msg, msgTwin)
		}else {
			pm.context.Lock(twinID)
			savedDesired  := savedTwin.Properties.Desired
			savedReported := savedTwin.Properties.Reported		
			newDesired := msgTwin.Properties.Desired
			newReported := msgTwin.Properties.Reported

			desiredProps := make(map[string]*types.PropertyValue)
			if newDesired != nil && len(newDesired) > 0 {
				if savedDesired == nil {
					pm.context.Unlock(twinID)
					pm.sendNotFoundPropMessage(msg, msgTwin)
					return nil
				}

				for key, _ := range newDesired {
					if value, exist := savedDesired[key]; !exist {
						desiredProps[key] = nil
						twinProperties:= &types.TwinProperties{
							Desired: desiredProps,
						}
						msgTwin.Properties = twinProperties
						msgTwin.State = savedTwin.State 
					
						pm.context.Unlock(twinID)

						pm.sendNotFoundPropMessage(msg, msgTwin)
						return nil
					}else {
						desiredProps[key] = value
					}
				}
			}
		
			reportedProps := make(map[string]*types.PropertyValue)
			if newReported != nil && len(newReported) > 0 {
				if savedReported == nil {
					pm.context.Unlock(twinID)
					pm.sendNotFoundPropMessage(msg, msgTwin)
					return nil
				}

				for key, _ := range newReported {
					if value, exist := savedReported[key]; !exist {
						reportedProps[key] = nil
						twinProperties:= &types.TwinProperties{
							Reported: reportedProps,
						}
						msgTwin.Properties = twinProperties
						msgTwin.State = savedTwin.State 
					
						pm.context.Unlock(twinID)
	
						//Not found this message.
						pm.sendNotFoundPropMessage(msg, msgTwin)
					
						return nil
					}else {
						reportedProps[key] = value
					}
				}
			}

			twinProperties:= &types.TwinProperties{
				Desired: desiredProps,
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
		}

		return nil
	})
}

// propWatchHandle: handle property watch.
// We don't know we send the property's update to who , since there are 2 reciever(cloud and edge/app)
// So reciever must call watch to recieve these twin properties 's update.
// If Properties is nil or no  properties in request message, we consider it to watch all properties of 
// this twin.
func (pm *PropertyModule) propWatchHandle (msg *model.Message ) error {
	return pm.handleMessage(msg, func(msg *model.Message, savedTwin, msgTwin *types.DigitalTwin) error{
		twinID := savedTwin.ID 
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
		}

		return nil
	})
}

//propSyncHandle sync reported property from device.
func (pm *PropertyModule) propSyncHandle (msg *model.Message ) error {
	var dgTwinMsg types.DGTwinMessage 

	if msg.GetSource() != "device" {
		klog.Infof("we just process the SYNC data from device.")
		return nil
	}

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
	for _, dgTwin := range dgTwinMsg.Twins	{
		if dgTwin == nil {
			klog.Infof("Twin is nil, Ignored")
			continue
		}
		if dgTwin.Properties == nil {
			klog.Infof("Twin Properties is nil, Ignored")
			continue
		}
		twinID := dgTwin.ID
		exist := pm.context.DGTwinIsExist(twinID)
		if exist{
			v, _ := pm.context.DGTwinList.Load(twinID)
			savedTwin, isDgTwinType  :=v.(*types.DigitalTwin)
			if !isDgTwinType {
				return errors.New("invalud digital twin type")
			}
 			if savedTwin == nil {
				return errors.New("savedTwin=nil, Unexpected error")
			}	
			if savedTwin.Properties == nil {
				klog.Infof("Unexpected error, savedTwin.Properties == nil")
				continue
			}
			//1. save the data.
			pm.context.Lock(twinID)
			savedReported := savedTwin.Properties.Reported	
			newReported := dgTwin.Properties.Reported
			if newReported != nil && len(newReported) > 0 {
				if savedReported == nil {
					pm.context.Unlock(twinID)
					pm.sendNotFoundPropMessage(msg, dgTwin)
					return nil
				}

				for key, value := range newReported {
					if _, ok := savedReported[key]; ok {
						savedReported[key] = value
					}
				}
			}
			pm.context.Unlock(twinID)
			//2. send response.
			twins := []*types.DigitalTwin{dgTwin}
			msgContent, err := types.BuildResponseMessage(types.RequestSuccessCode, "SYNC Success", twins)
			if err != nil {
				return err
			}
			pm.context.SendResponseMessage(msg, msgContent)
			
			//3. Check the cache event, if these property is cached, then send SYNC to target
			pm.context.RangeWatchCache(func(key, value interface{}) bool {
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
			})
		}
	}

	return nil
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
		twinID := dgTwin.ID
		exist := pm.context.DGTwinIsExist(twinID)
		if !exist {
			// Device has not created yet.
			twins := []*types.DigitalTwin{dgTwin}
			msgContent, err := types.BuildResponseMessage(types.NotFoundCode, "Twin Not found", twins)
			if err != nil {
				return err
			}
			pm.context.SendResponseMessage(msg, msgContent)
		}else{
			v, _ := pm.context.DGTwinList.Load(twinID)
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

// propResponseHandle: handle all response.
func (pm *PropertyModule) propResponseHandle (msg *model.Message ) error {
	var resp types.DGTwinResponse

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
	}

	//Success
	pm.context.SendToModule(types.DGTWINS_MODULE_COMM, msg)

	return nil
}

func (pm *PropertyModule) sendNotFoundPropMessage(msg *model.Message, msgTwin *types.DigitalTwin) error {
	twins := []*types.DigitalTwin{msgTwin}
	msgContent, err := types.BuildResponseMessage(types.NotFoundCode, "twin No property/No this property", twins)
	if err != nil {
		return err
	}
			
	pm.context.SendResponseMessage(msg, msgContent)

	return nil
}
