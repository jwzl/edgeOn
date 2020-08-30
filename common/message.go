package common

import (
	"time"
	"errors"
	"strings"
	"encoding/json"
	"github.com/jwzl/wssocket/model"
)

const (

	//BadRequestCode sucess
	RequestSuccessCode= 200 
	//device is online.
	OnlineCode= 600 
	//Close Watch code.
	CloseWatchCode= 700 
	//BadRequestCode bad request
	BadRequestCode = 400
	//NotFoundCode device not found
	NotFoundCode = 404
	//ConflictCode version conflict
	ConflictCode = 409
	//InternalErrorCode server internal error
	InternalErrorCode = 500
	//DeviceFound device is found. 
	DeviceFound		= 600
	// DeviceNotReady device is not ready.
	DeviceNotReady	= 601
	
	//twin's verb
	DGTWINS_OPS_CREATE		= "Create"
	DGTWINS_OPS_UPDATE		= "Update"
	DGTWINS_OPS_DELETE		= "Delete"
	DGTWINS_OPS_GET			= "Get"
	DGTWINS_OPS_List		= "List"
	DGTWINS_OPS_RESPONSE	= "Response"
	DGTWINS_OPS_WATCH		= "Watch"
	DGTWINS_OPS_SYNC		= "Sync"
	DGTWINS_OPS_DETECT		= "Detect"
	DGTWINS_OPS_KEEPALIVE		= "Keepalive"

	//State
	DGTWINS_STATE_CREATED	= "created"	
	DGTWINS_STATE_ONLINE	= "online"	
	DGTWINS_STATE_OFFLINE	= "offline"

	// Resource
	DGTWINS_RESOURCE_EDGE	="edge"	
	DGTWINS_RESOURCE_TWINS	="twins"
	DGTWINS_RESOURCE_PROPERTY	="property"
	DGTWINS_RESOURCE_DEVICE	="device"

	HubModuleName	=  "edge/hub"
	CloudName		= "cloud"
	EdgeAppName		= "edge/app"
	TwinModuleName	= "edge/dgtwin"
	BusModuleName	= "edge/eventbus"
	DeviceName		= "device"
)

//Create/update/Delete/Get twins message format
type TwinMessage struct{	
	Twins  []DigitalTwin 	`json:"twins"`
}

// Response message format
type TwinResponse struct{
	Code   int    			`json:"code"`
	Reason string 			`json:"reason,omitempty"`
	Twins  []DigitalTwin		`json:"twins,omitempty"`
}

/*
* Device Message.
*/
// message send to device or from device to sync.
type DeviceMessage struct{	
	Twin  DeviceTwin	 	`json:"twin"`
}

// response message from device.
type DeviceResponse struct{
	// response code.
	Code   string    			`json:"code"`
	// fail reson
	Reason string 				`json:"reason,omitempty"`
	Twin  DeviceTwin			`json:"twin,omitempty"`
}

// GetDesiredProperties
func GetDesiredProperties(twin *DeviceTwin) []TwinProperty {
	return twin.Properties.Desired
}

// GetReportedProperties
func GetReportedProperties(twin *DeviceTwin) []TwinProperty {
	return twin.Properties.Reported
}

// GetPropertyValue
func GetPropertyValue(props []TwinProperty, name string) *TwinProperty {
	for key, _ := range props {
		if name == props[key].Name {
			return &props[key]
		}
	}

	return nil
}

// BuildResponseMessage
func BuildResponseMessage(code int, reason string, twins []DigitalTwin) ([]byte, error){
	resp := &TwinResponse{
		Code: code,
		Reason: reason,
		Twins: twins,
	}

	resultJSON, err := json.Marshal(resp)

	return resultJSON, err
}

// UnMarshalResponseMessage
func UnMarshalResponseMessage(msg *model.Message)(*TwinResponse, error){
	var rspMsg TwinResponse
	content, ok := msg.Content.([]byte)
	if !ok {
		return nil, errors.New("invaliad message content")
	}

	err := json.Unmarshal(content, &rspMsg)
	if err != nil {
		return nil, err
	}

	return &rspMsg, nil
}

//UnMarshalTwinMessage.
func UnMarshalTwinMessage(msg *model.Message)(*TwinMessage, error){
	var twinMsg TwinMessage

	content, ok := msg.Content.([]byte)
	if !ok {
		return nil, errors.New("invaliad message content")
	}

	err := json.Unmarshal(content, &twinMsg)
	if err != nil {
		return nil, err
	}

	return &twinMsg, nil
}

// BuildTwinMessage
func BuildTwinMessage(twins []DigitalTwin) ([]byte, error){
	twinMsg := &TwinMessage{
		Twins: twins,
	}

	resultJSON, err := json.Marshal(twinMsg)

	return resultJSON, err
}

func BuildModelMessage(source string, target string, operation string, resource string, content interface{}) *model.Message {
	now := time.Now().UnixNano() / 1e6
	
	//Header
	msg := model.NewMessage("")
	msg.BuildHeader("", now)

	//Router
	msg.BuildRouter(source, "", target, resource, operation)
	
	//content
	switch content.(type){
	case []byte:
		msg.Content = content
	default:
		bytes, _ := json.Marshal(content)
		//if err == nil {
			msg.Content = bytes
		//}
	}	
	
	return msg
}


// GetTwinID
func GetTwinID(msg *model.Message) string {
	var twins  []DigitalTwin
	operation := msg.GetOperation()
	target := msg.GetTarget()
	
	content, ok := msg.Content.([]byte)
	if !ok {
		return ""
	}
	
	if strings.Contains(target, DGTWINS_RESOURCE_DEVICE) {
		if strings.Compare(DGTWINS_OPS_RESPONSE, operation) == 0 {
			var resp DeviceResponse
			
			err := json.Unmarshal(content, &resp)
			if err != nil {
				return ""
			}
			
			return resp.Twin.ID
		}else {
			var devMsg DeviceMessage
		
			err := json.Unmarshal(content, &devMsg)
			if err != nil {
				return ""
			}
			
			return devMsg.Twin.ID	
		}
	}else{
		//twin message.
		if strings.Compare(DGTWINS_OPS_RESPONSE, operation) == 0 {
			var resp TwinResponse

			err := json.Unmarshal(content, &resp)
			if err != nil {
				return ""
			}

			twins = resp.Twins	
		}else {
			var dgTwinMsg TwinMessage

			err := json.Unmarshal(content, &dgTwinMsg)
			if err != nil {
				return ""
			}	
		
			twins = dgTwinMsg.Twins	
		}
	}

	

	for _, dgTwin := range twins {
		return dgTwin.ID
	}	

	return ""
}

// Build device message.
func BuildDeviceMessage(twin *DeviceTwin) ([]byte, error){
	DeviceMsg := &DeviceMessage{
		Twin:	*twin,		
	}

	return json.Marshal(DeviceMsg)
}

// UnMarshal the device message.
func UnMarshalDeviceMessage(msg *model.Message)(*DeviceMessage, error){
	var deviceMsg DeviceMessage

	content, ok := msg.Content.([]byte)
	if !ok {
		return nil, errors.New("invaliad message content")
	}

	err := json.Unmarshal(content, &deviceMsg)
	if err != nil {
		return nil, err
	}

	return &deviceMsg, nil
}

// Build device response message.
func BuildDeviceResponseMessage(code string, reason string, twin *DeviceTwin) ([]byte, error){
	respMsg := &DeviceResponse{
		Code:	code,
		Reason:	reason,
		Twin:	*twin,	
	}

	resultJSON, err := json.Marshal(respMsg)
	return resultJSON, err
}

// UnMarshal the device response message.
func UnMarshalDeviceResponseMessage(msg *model.Message)(*DeviceResponse, error){
	var respMsg DeviceResponse

	content, ok := msg.Content.([]byte)
	if !ok {
		return nil, errors.New("invaliad message content")
	}

	err := json.Unmarshal(content, &respMsg)
	if err != nil {
		return nil, err
	}

	return &respMsg, nil
}

type EdgeInfo struct{
	EdgeID		string	`json:"edgeid"`
	EdgeName	string	`json:"edgename,omitempty"`
	Description	string	`json:"description,omitempty"`
}
