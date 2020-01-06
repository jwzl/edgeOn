package types

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

	//twin's verb
	DGTWINS_OPS_UPDATE		= "Update"
	DGTWINS_OPS_DELETE		= "Delete"
	DGTWINS_OPS_GET			= "Get"
	DGTWINS_OPS_RESPONSE	= "Response"
	DGTWINS_OPS_WATCH		= "Watch"
	DGTWINS_OPS_CREATE		= "Create"
	DGTWINS_OPS_SYNC		= "Sync"

	//State
	DGTWINS_STATE_ONLINE	="online"	
	DGTWINS_STATE_OFFLINE	="offline"

	// Resource
	DGTWINS_RESOURCE_TWINS	="twins"
	DGTWINS_RESOURCE_PROPERTY	="property"
	DGTWINS_RESOURCE_DEVICE	="device"

	HubModuleName	=  "edge/hub"
	CloudName		= "cloud"
	EdgeAppName		= "edge/app"
	TwinModuleName	= "edge/dgtwin"
)


//Create/update/Delete/Get twins message format
type DGTwinMessage struct{
	Action	string			`json:"action,omitempty"`
	Timestamp int64  		`json:"timestamp"`		
	Twins  []*DigitalTwin 	`json:"twins"`
}

// Response message format
type DGTwinResponse struct{
	Code   int    			`json:"code"`
	Reason string 			`json:"reason,omitempty"`
	Timestamp int64  		`json:"timestamp"`	
	Twins  []*DigitalTwin	`json:"twins,omitempty"`
}

// BuildResponseMessage
func BuildResponseMessage(code int, reason string, twins  []*DigitalTwin) ([]byte, error){
	now := time.Now().UnixNano() / 1e6

	resp := &DGTwinResponse{
		Code: code,
		Reason: reason,
		Timestamp: now,
		Twins: twins,
	}

	resultJSON, err := json.Marshal(resp)

	return resultJSON, err
}

// UnMarshalResponseMessage
func UnMarshalResponseMessage(msg *model.Message)(*DGTwinResponse, error){
	var rspMsg DGTwinResponse
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

// BuildTwinMessage
func BuildTwinMessage(action string, twins []*DigitalTwin) ([]byte, error){
	now := time.Now().UnixNano() / 1e6

	twinMsg := &DGTwinMessage{
		Action: action,
		Timestamp: now,
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
	msg.Content = content
	
	return msg
}


// GetTwinID
func GetTwinID(msg *model.Message) string {
	var twins  []*DigitalTwin
	operation := msg.GetOperation()
	
	content, ok := msg.Content.([]byte)
	if !ok {
		return ""
	}

	if strings.Compare(DGTWINS_OPS_RESPONSE, operation) == 0 {
		var resp DGTwinResponse

		err := json.Unmarshal(content, &resp)
		if err != nil {
			return ""
		}

		twins = resp.Twins	
	}else {
		var dgTwinMsg DGTwinMessage

		err := json.Unmarshal(content, &dgTwinMsg)
		if err != nil {
			return ""
		}	
		
		twins = dgTwinMsg.Twins	
	}

	for _, dgTwin := range twins {
		if dgTwin == nil {
			continue
		}

		return dgTwin.ID
	}	

	return ""
}
