package dtmodule

import (
	"sync"
	"time"
	"testing"
	"strings"
	"encoding/json"
	"github.com/jwzl/wssocket/model"
	"github.com/jwzl/beehive/pkg/core/context"
	"github.com/jwzl/edgeOn/digitaltwin/pkg/types"
	"github.com/jwzl/edgeOn/digitaltwin/pkg/dtcontext"	
)

type PropertyTest struct {
	context			*dtcontext.DTContext	
	module			*PropertyModule	
	commChan		chan interface{}
}

func NewPropertyTest() *PropertyTest {
	ctx := context.GetContext(context.MsgCtxTypeChannel)
	dtcontext := dtcontext.NewDTContext(ctx)
	propModule := NewPropertyModule()
	comm := make(chan interface{}, 128)

	return &PropertyTest{
		context: dtcontext,
		module:	propModule,
		commChan: comm,
	}
}

func (pt *PropertyTest) Start() {
	propModule := pt.module
	pt.context.CommChan["comm"] = pt.commChan
	pt.context.RegisterDTModule(propModule)

	// Start this module.
	go propModule.Start()
}

func (pt *PropertyTest) StroeTwin(dgTwin *types.DigitalTwin){
	if  dgTwin != nil {
		twinID := dgTwin.ID
		pt.context.DGTwinList.Store(twinID, dgTwin)
		var deviceMutex	sync.Mutex
		pt.context.DGTwinMutex.Store(twinID, &deviceMutex)
	}
}

func (pt *PropertyTest) LoadTwin(twinID string)  *types.DigitalTwin {
	v, exist := pt.context.DGTwinList.Load(twinID)
	if !exist {
		return nil
	}
	savedTwin, isDgTwinType  :=v.(*types.DigitalTwin)
	if !isDgTwinType {
		return nil
	}

	return savedTwin
}

func TestNewPropertyModule(t *testing.T){
	propModule := NewPropertyModule()
	if propModule == nil {
		t.Errorf("failed to create property module.")
	}
}

// TestPropUpdateHandle test property update.
func TestPropUpdateHandle(t *testing.T){
	pt := NewPropertyTest()
	pt.Start()	
	t.Log("Start test PropUpdateHandle ")

	dgTwin := &types.DigitalTwin{
		ID:	"dev001",
		Name:	"sensor0",
		Description: "None",
		State: "offline",
	}

	// Store the twin
	pt.StroeTwin(dgTwin)

	//update.
	err := pt.propertyDoHandle("dev001", "reboot", types.DGTWINS_OPS_UPDATE, &types.PropertyValue{Value: "1"})
	if err != nil {
		t.Fatal("Update error ")
	}
	time.Sleep(10 * time.Millisecond)

	//Load the twin.
	savedTwin :=pt.LoadTwin("dev001")
	if savedTwin == nil {
		t.Fatal("error twin by LoadTwin ")
	}

	if savedTwin.Properties == nil {
		t.Fatal("update failed, oldTwin.Properties is empty.")
	}

	savedDesired  := savedTwin.Properties.Desired

	if val, exist := savedDesired["reboot"]; exist {
		a := val.Value.(string)
		if a != "1" {
			t.Fatal("update value is error.")
		}
	}else {
		t.Fatal("No reboot property.")
	}	
	t.Log("property update success. ")

	// Check the response 
	v, ok := <- pt.commChan
	if !ok {
		t.Fatal("Channel has closed..")
	}
	response := GetDTResponse(v)
	if response == nil {
		t.Fatal("Response error format.")
	}

	if response.Code !=types.RequestSuccessCode {
		t.Fatal("Response err")
	}
	t.Log("Response okay. ")

	//Check the message to device.
	v, ok = <- pt.commChan
	if !ok {
		t.Fatal("Channel has closed..")
	}
	twins := GetTwins(v)
	if twins == nil {
		t.Fatal("No twins")
	}

	dgTwin = twins[0]
	if dgTwin == nil {
		t.Fatal("No twins")
	}

	if dgTwin.ID != "dev001" {
		t.Fatal("error message")
	}

	if dgTwin.Properties == nil || dgTwin.Properties.Desired == nil ||
			len(dgTwin.Properties.Desired) < 1 {
		t.Fatal("no property")
	}

	desired := dgTwin.Properties.Desired
	if val, exist :=desired["reboot"]; !exist {
		t.Fatal("error update")
	}else {
		if val.Value != "1" {
			t.Fatal("error update")
		}
	}
	t.Log("device message is okay. ")

	pt.context.StopModule("property")
}

func GetDTResponse(v interface{})*types.DGTwinResponse{
	message, isMsgType := v.(*model.Message )
	if !isMsgType {
		return nil
	}

	content, ok := message.Content.([]byte)
	if !ok {
		return nil
	}

	var resp types.DGTwinResponse
	err := json.Unmarshal(content, &resp)
	if err != nil {
		return nil
	}

	return &resp
}

func GetDTMessage(v interface{})*types.DGTwinMessage{
	message, isMsgType := v.(*model.Message )
	if !isMsgType {
		return nil
	}

	content, ok := message.Content.([]byte)
	if !ok {
		return nil
	}

	var dgTwinMsg types.DGTwinMessage
	err := json.Unmarshal(content, &dgTwinMsg)
	if err != nil {
		return nil
	}

	return &dgTwinMsg	
}

func GetTwins(v interface{})[]*types.DigitalTwin{
	var twins  []*types.DigitalTwin
	message, isMsgType := v.(*model.Message )
	if !isMsgType {
		return nil
	}
	
	operation := message.GetOperation()

	if strings.Compare(types.DGTWINS_OPS_RESPONSE, operation) == 0 {
		resp := GetDTResponse(v)
		if resp ==nil {
			return nil
		}
		twins = resp.Twins
	}else {
		dgTwinMsg :=GetDTMessage(v)
		if dgTwinMsg ==nil {
			return nil
		}
		twins = dgTwinMsg.Twins
	}

	return twins
}  

func (pt *PropertyTest) propertyDoHandle(twinID, propName, action string, value *types.PropertyValue) error{
	props := &types.TwinProperties{
		Desired: map[string]*types.PropertyValue{
			propName:	value,		
		},
	}
	twin := &types.DigitalTwin{
		ID:	twinID,
		Properties: props, 
	}

	twins := []*types.DigitalTwin{twin}
	bytes, err := types.BuildTwinMessage(action, twins)
	if err != nil {
		return err
	}
		
	modelMsg := pt.context.BuildModelMessage("edge/app", types.MODULE_NAME, 
							action, types.DGTWINS_MODULE_PROPERTY, bytes)

	pt.context.SendToModule(types.DGTWINS_MODULE_PROPERTY, modelMsg) 

	return nil
}

func TestPropDeleteHandle(t *testing.T){
	pt := NewPropertyTest()
	pt.Start()	
	t.Log("Start test PropDeleteHandle ")

	props := &types.TwinProperties{
		Desired: map[string]*types.PropertyValue{
			"on/off":	&types.PropertyValue{Value: "0"},
			"reboot":	&types.PropertyValue{Value: "1"},
			"holdon":	&types.PropertyValue{Value: "2"},		
		},
	}
	dgTwin := &types.DigitalTwin{
		ID:	"dev001",
		Name:	"sensor0",
		Description: "None",
		State: "offline",
		Properties: props, 
	}

	// Store the twin
	pt.StroeTwin(dgTwin)

	//Delete.
	err := pt.propertyDoHandle("dev001", "reboot", types.DGTWINS_OPS_DELETE, nil)
	if err != nil {
		t.Fatal("Delete error ")
	}
	time.Sleep(5 * time.Millisecond)
	//Load the twin.
	savedTwin :=pt.LoadTwin("dev001")
	if savedTwin == nil {
		t.Fatal("error twin by LoadTwin ")
	}

	if savedTwin.Properties == nil {
		t.Fatal("failed, oldTwin.Properties is empty.")
	}

	savedDesired  := savedTwin.Properties.Desired
	if savedDesired  == nil {
		t.Fatal("Desired is nil.")
	}

	if _, exist := savedDesired["reboot"]; exist {
		t.Fatal("Delete error.")
	}
	t.Log("Delete sucessful. ")
	
	//check the reponse.
	v, ok := <- pt.commChan
	if !ok {
		t.Fatal("Channel has closed..")
	}
	response := GetDTResponse(v)
	if response == nil {
		t.Fatal("Response error format.")
	}

	if response.Code !=types.RequestSuccessCode {
		t.Fatal("Response err")
	}
	t.Log("Response okay. ")

	//Check the message to device.
	v, ok = <- pt.commChan
	if !ok {
		t.Fatal("Channel has closed..")
	}
	twins := GetTwins(v)
	if twins == nil {
		t.Fatal("No twins")
	}

	dgTwin = twins[0]
	if dgTwin == nil {
		t.Fatal("No twins")
	}

	if dgTwin.ID != "dev001" {
		t.Fatal("error message")
	}

	if dgTwin.Properties == nil || dgTwin.Properties.Desired == nil ||
			len(dgTwin.Properties.Desired) < 1 {
		t.Fatal("no property")
	}

	desired := dgTwin.Properties.Desired
	if _, exist :=desired["reboot"]; !exist {
		t.Fatal("error delete")
	}
	t.Log("delete device message is okay. ")

	pt.context.StopModule("property")
}


func TestPropGetHandle(t *testing.T){
	pt := NewPropertyTest()
	pt.Start()	
	t.Log("Start test PropGetHandle ")

	props := &types.TwinProperties{
		Desired: map[string]*types.PropertyValue{
			"on/off":	&types.PropertyValue{Value: "0"},
			"reboot":	&types.PropertyValue{Value: "1"},
			"holdon":	&types.PropertyValue{Value: "2"},		
		},
	}
	dgTwin := &types.DigitalTwin{
		ID:	"dev001",
		Name:	"sensor0",
		Description: "None",
		State: "offline",
		Properties: props, 
	}

	// Store the twin
	pt.StroeTwin(dgTwin)

	//Get.
	err := pt.propertyDoHandle("dev001", "reboot", types.DGTWINS_OPS_GET, nil)
	if err != nil {
		t.Fatal("Get error ")
	}

	//check the reponse.
	v, ok := <- pt.commChan
	if !ok {
		t.Fatal("Channel has closed..")
	}
	response := GetDTResponse(v)
	if response == nil {
		t.Fatal("Response error format.")
	}

	if response.Code !=types.RequestSuccessCode {
		t.Fatal("Response err")
	}
	
	twins := response.Twins
	if twins == nil {
		t.Fatal("twins is empty.")
	}
	
	twin := twins[0]
	if twin == nil {
		t.Fatal("twin is empty.")
	}

	property := twin.Properties
	if property == nil || len(property.Desired) != 1 {
		t.Fatal("property is nil.")
	}
	if val, exist := property.Desired["reboot"]; !exist {
		t.Fatal("property is not exist.")
	}else {
		if val.Value != "1" {
			t.Fatal("Get error.")		
		}
	}
	t.Log("Response okay. ")

	//Check  the error response
	err = pt.propertyDoHandle("dev001", "fuck", types.DGTWINS_OPS_GET, nil)
	if err != nil {
		t.Fatal("Get error ")
	}

	//check the reponse.
	v, ok = <- pt.commChan
	if !ok {
		t.Fatal("Channel has closed..")
	}
	response = GetDTResponse(v)
	if response == nil {
		t.Fatal("Response error format.")
	}

	if response.Code !=types.NotFoundCode {
		t.Fatal("property founded")
	}
	t.Log("Check okay. ")

	pt.context.StopModule("property")	
}
