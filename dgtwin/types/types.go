package types

import (
	"github.com/jwzl/wssocket/model"
)

const (
	MODULE_NAME = "edge/dgtwin"
	//module name
	DGTWINS_MODULE_TWINS	= "twins"
	DGTWINS_MODULE_PROPERTY	= "property"
	DGTWINS_MODULE_COMM	= "comm"

	DGTWINS_MSG_TIMEOUT = 1*60		//5s 
)

type DTMessage struct {
	Msg      	*model.Message
	Source 		string
	Target 		string
	Operation   string
}

type WatchEvent	struct {
	MsgID		string
	TwinID		string
	// message's source 
	Source 		string	
	// message's resource
	Resource 	string
	//Slice for watch's property/attributes.  
	List		[]string	
}

func CreateWatchEvent(msgID, twinID, source, resource string) *WatchEvent {
	return &WatchEvent{
		MsgID:	msgID,
		TwinID:	twinID,
		Source: source,
		Resource: resource,
		List: 	make([]string, 0),
	}
}
//Back-end opertaion
//1. Retrieve digital twin by ID.
//2. Partially update digital twin. 
//3. Replace desired properties.
//4. Replace metadata
//5. Receive twin notifications.

//Device operations
//1. Retrieve device twin
//2. Partially update reported properties
//3. Observe desired properties
