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

	DGTWINS_MSG_TIMEOUT = 5		//5s 
)
// DigitalTwin is a digital description about things in physical world. If you want to do something
// for things in physical world, you just need to access the digitaltwin. 
// you can modify a things's property's value as desired state, and tings can report the property value as 
// reported state.   
type DigitalTwin struct {
	// device id	
	ID		string		`json:"id"`	
	//device name
	Name	string 		`json:"name,omitempty"`
	// device description
	Description		string		`json:"description,omitempty"`
	// device state
	State	string 		`json:"state,omitempty"`
	// device last state
	LastState	string	`json:"laststate,omitempty"`
	// device metadata  
	MetaData	map[string]string	`json:"metadata,omitempty"`
	//all properties
	Properties	TwinProperties			`json:"properties,omitempty"`	
}

// all Desired and Reported are in TwinProperties.
type TwinProperties struct {
	Desired  map[string]PropertyValue		`json:"desired,omitempty"`
	Reported map[string]PropertyValue			`json:"reported,omitempty"`	
}

type PropertyValue struct {
	// the value type is default as string, if you need other type
	// please insert Type:xxx into metadata map.	
	Value	interface{}		`json:"value"`
	// you can add other key:value for the property
	MetaData	map[string]interface{}		`json:"metadata,omitempty"`
}

type DTMessage struct {
	Msg      	*model.Message
	Source 		string
	Target 		string
	Operation   string
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
