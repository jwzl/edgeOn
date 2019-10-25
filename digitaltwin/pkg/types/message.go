package types

const {

	//BadRequestCode sucess
	RequestSuccessCode= 200 
	//BadRequestCode bad request
	BadRequestCode = 400
	//NotFoundCode device not found
	NotFoundCode = 404
	//ConflictCode version conflict
	ConflictCode = 409
	//InternalErrorCode server internal error
	InternalErrorCode = 500

	DGTWINS_OPS_TWINSUPDATE	= "TwinsUpdate"
	DGTWINS_OPS_TWINSUPDATE	= "TwinsDelete"
	DGTWINS_OPS_TWINSGET	= "TwinsGet"
	DGTWINS_OPS_RESPONSE	= "Response"

	//Device
	DGTWINS_OPS_DEVCREATE	= "Create"
	DGTWINS_OPS_TWINSUPDATE	= "TwinsDelete"
	DGTWINS_OPS_TWINSGET	= "TwinsGet"
	DGTWINS_OPS_RESPONSE	= "Response"
}



//Create/update/Delete/Get twins message format
type DGTwinMessage struct{
	Action	string			`json:"action,omitempty"`
	Timestamp int64  		`json:"timestamp"`		
	Twins  []DigitalTwin 	`json:"twins"`
}

// Response message format
type DGTwinResponse struct{
	Code   int    			`json:"code"`
	Reason string 			`json:"reason,omitempty"`
	Timestamp int64  		`json:"timestamp"`	
	Twins  []DigitalTwin	`json:"twins,omitempty"`
}


func BuildResponseMessage(code int, reason string, twins  []DigitalTwin) ([]byte error){
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
