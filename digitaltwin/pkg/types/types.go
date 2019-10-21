package types

type DeviceTwin struct {
	ID		string		`json:"id,omitempty"`	
	Name	string 		`json:"name,omitempty"`
	Description		string		`json:"description,omitempty"`
	State	string 		`json:"state,omitempty"`
	LastState	string	`json:"laststate,omitempty"`
	MetaData	[string]string	`json:metadata,omitempty`
	Properties	DeviceProperties				
}

type DeviceProperties struct {
	Desired  [string]PropertyValue		`json:"desired,omitempty"`
	Reported [string]PropertyValue			`json:"reported,omitempty"`	
}

type PropertyValue struct {
	// the value type is default as string, if you need other type
	// please insert Type:xxx into metadata map.	
	Value	string		`json:"value"`
	// you can add other key:value for the property
	MetaData	[string]string		`json:"metadata,omitempty"`
}
