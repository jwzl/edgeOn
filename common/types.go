package common

const (

	// property value type.
	TWIN_PROP_VALUE_TYPE_CHAR	= "char"
	TWIN_PROP_VALUE_TYPE_UINT8	= "int8"
	TWIN_PROP_VALUE_TYPE_UINT16	= "int16"
	TWIN_PROP_VALUE_TYPE_UINT32	= "int32"
	TWIN_PROP_VALUE_TYPE_UINT64	= "int64"
	TWIN_PROP_VALUE_TYPE_STRING	= "string"
	TWIN_PROP_VALUE_TYPE_BYTES	= "bytes"
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
	MetaData	map[string]*MetaType	`json:"metadata,omitempty"`
	//all properties
	Properties	TwinProperties			`json:"properties,omitempty"`	
}

// all Desired and Reported are in TwinProperties.
type TwinProperties struct {
	Desired  map[string]*TwinProperty		`json:"desired,omitempty"`
	Reported map[string]*TwinProperty			`json:"reported,omitempty"`	
}

/*type PropertyValue struct {
	// the value type is default as string, if you need other type
	// please insert Type:xxx into metadata map.	
	Value	interface{}		`json:"value"`
	// you can add other key:value for the property
	MetaData	map[string]interface{}		`json:"metadata,omitempty"`
}*/

// DeviceTwin is a digital description about things in physical world. If you want to do something
// for things in physical world, you just need to access the digitaltwin. 
// you can modify a things's property's value as desired state, and tings can report the property value as 
// reported state. 
type DeviceTwin struct{
	// device id	
	ID		string					`json:"id"`
	//device name
	Name	string 					`json:"name,omitempty"`
	// device description
	Description		string			`json:"description,omitempty"`
	// device state
	State	string 					`json:"state,omitempty"`
	// device last state
	LastState	string				`json:"laststate,omitempty"`
	// device metadata  
	MetaData	[]MetaType			`json:"metadata,omitempty"`

	//device properties
	Properties	DeviceTwinProperties		`json:"properties,omitempty"`
}

// all Desired and Reported are in TwinProperties.
type DeviceTwinProperties struct {
	Desired  []TwinProperty			`json:"desired,omitempty"`
	Reported []TwinProperty			`json:"reported,omitempty"`	
}

type TwinProperty struct {
	Name	string 					`json:"name,omitempty"`
	Value	[]byte					`json:"value"`
	Type	string 					`json:"type,omitempty"`
	/* property meta data.*/
	MetaData	[]MetaType			`json:"metadata,omitempty"`
}

type MetaType struct{
	Name	string 					`json:"name,omitempty"`
	Value	string 					`json:"value,omitempty"`
}
