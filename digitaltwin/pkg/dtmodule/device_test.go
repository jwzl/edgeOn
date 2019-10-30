package dtmodule

import (
	"sync"
	"time"
	"testing"
	"encoding/json"
	"github.com/jwzl/wssocket/model"
	"github.com/jwzl/beehive/pkg/core/context"
	"github.com/jwzl/edgeOn/digitaltwin/pkg/types"
	"github.com/jwzl/edgeOn/digitaltwin/pkg/dtcontext"	
)

func TestNewDeviceModule(t *testing.T) {
	deviceModule := NewDeviceModule("device")
	if deviceModule == nil {
		t.Fatal("Failed to create device module.")
	}
}

//TestCreateTwin
func TestCreateTwin(t *testing.T) {
	ctx := context.GetContext(context.MsgCtxTypeChannel)
	dtcontext := dtcontext.NewDTContext(ctx)
	deviceModule := NewDeviceModule("device")
	comm := make(chan interface{}, 128)
	heartBeat := make(chan interface{}, 128)

	deviceModule.InitModule(dtcontext, comm, heartBeat, nil)
	t.Log("Start test CreateTwin")

	dgTwin := types.DigitalTwin{
		ID:	"dev001",
		Name:	"sensor0",
		Description: "None",
		State: "offline",
	}
	twins := []types.DigitalTwin{dgTwin}
	bytes, err := types.BuildTwinMessage(types.DGTWINS_OPS_TWINSUPDATE, twins)
	if err == nil {
		modelMsg := dtcontext.BuildModelMessage("edge/app", types.MODULE_NAME, 
							types.DGTWINS_OPS_TWINSUPDATE, types.DGTWINS_MODULE_TWINS, bytes)
		comm <- modelMsg
		heartBeat <- "ping"
	}

	t.Run("Create", func(t *testing.T){
		go deviceModule.Start()
		time.Sleep(10 * time.Millisecond)
		
		_, exist := dtcontext.DGTwinList.Load("dev001")
		if !exist {
			t.Errorf("No Such twin!")
		}
	})

	heartBeat <- "stop"
}

// TestUpdateTwin
func TestUpdateTwin(t *testing.T) {
	ctx := context.GetContext(context.MsgCtxTypeChannel)
	dtcontext := dtcontext.NewDTContext(ctx)
	deviceModule := NewDeviceModule("device")
	comm := make(chan interface{}, 128)
	heartBeat := make(chan interface{}, 128)

	deviceModule.InitModule(dtcontext, comm, heartBeat, nil)
	t.Log("Start test UpdateTwin")

	dgTwin := types.DigitalTwin{
		ID:	"dev001",
		Name:	"sensor0",
		Description: "None",
		State: "offline",
	}
	// Store the twin
	dtcontext.DGTwinList.Store("dev001", &dgTwin)
	var deviceMutex	sync.Mutex
	dtcontext.DGTwinMutex.Store("dev001", &deviceMutex)

	newTwin := types.DigitalTwin{
		ID:	"dev001",
		Name:	"sensor1",
		Description: "",
		State: "offline",
	}
	twins := []types.DigitalTwin{newTwin}
	bytes, err := types.BuildTwinMessage(types.DGTWINS_OPS_TWINSUPDATE, twins)
	if err == nil {
		modelMsg := dtcontext.BuildModelMessage("edge/app", types.MODULE_NAME, 
							types.DGTWINS_OPS_TWINSUPDATE, types.DGTWINS_MODULE_TWINS, bytes)
		comm <- modelMsg
		heartBeat <- "ping"
	}
	
	t.Run("Update", func(t *testing.T){
		go deviceModule.Start()
		time.Sleep(10 * time.Millisecond)
		
		v, exist := dtcontext.DGTwinList.Load("dev001")
		if !exist {
			t.Errorf("No Such twin!")
		}
		oldTwin, isDgTwinType  :=v.(*types.DigitalTwin)
		if !isDgTwinType {
			t.Errorf("invalud digital twin type")
		}

		if oldTwin.Name != newTwin.Name {
			t.Errorf("Name err %s != %s", oldTwin.Name, newTwin.Name)
		}
		if oldTwin.Description != dgTwin.Description {
			t.Errorf("Name err %s != %s", oldTwin.Name, newTwin.Name)
		}
	})

	heartBeat <- "stop"
}

func TestDeleteTwin(t *testing.T) {
	ctx := context.GetContext(context.MsgCtxTypeChannel)
	dtcontext := dtcontext.NewDTContext(ctx)
	deviceModule := NewDeviceModule("device")
	comm := make(chan interface{}, 128)
	heartBeat := make(chan interface{}, 128)

	deviceModule.InitModule(dtcontext, comm, heartBeat, nil)
	t.Log("Start test DeleteTwin")

	dgTwin := types.DigitalTwin{
		ID:	"dev001",
		Name:	"sensor0",
		Description: "None",
		State: "offline",
	}
	// Store the twin
	dtcontext.DGTwinList.Store("dev001", &dgTwin)
	var deviceMutex	sync.Mutex
	dtcontext.DGTwinMutex.Store("dev001", &deviceMutex)

	newTwin := types.DigitalTwin{
		ID:	"dev001",
	}
	twins := []types.DigitalTwin{newTwin}
	bytes, err := types.BuildTwinMessage(types.DGTWINS_OPS_TWINDELETE, twins)
	if err == nil {
		modelMsg := dtcontext.BuildModelMessage("edge/app", types.MODULE_NAME, 
							types.DGTWINS_OPS_TWINDELETE, types.DGTWINS_MODULE_TWINS, bytes)
		comm <- modelMsg
		heartBeat <- "ping"
	}
	
	t.Run("Delete", func(t *testing.T){
		go deviceModule.Start()
		time.Sleep(10 * time.Millisecond)
		
		_, exist := dtcontext.DGTwinList.Load("dev001")
		if exist {
			t.Errorf("Not delete this!")
		}

		_, exist = dtcontext.DGTwinMutex.Load("dev001")
		if exist {
			t.Errorf("Not delete this!")
		}
	})

	heartBeat <- "stop"
}

func TestGetTwin(t *testing.T) {
	ctx := context.GetContext(context.MsgCtxTypeChannel)
	dtcontext := dtcontext.NewDTContext(ctx)
	dtcontext.CommChan["comm"] = make(chan interface{}, 128)
	dtcontext.HeartBeatChan["comm"] = make(chan interface{}, 128)
	deviceModule := NewDeviceModule("device")
	comm := make(chan interface{}, 128)
	heartBeat := make(chan interface{}, 128)

	deviceModule.InitModule(dtcontext, comm, heartBeat, nil)
	t.Log("Start test DeleteTwin")

	dgTwin := types.DigitalTwin{
		ID:	"dev001",
		Name:	"sensor0",
		Description: "None",
		State: "offline",
	}
	dgTwin2 := types.DigitalTwin{
		ID:	"dev002",
		Name:	"sensor1",
		Description: "None",
		State: "offline",
	}
	// Store the twin
	dtcontext.DGTwinList.Store("dev001", &dgTwin)
	var deviceMutex	sync.Mutex
	dtcontext.DGTwinMutex.Store("dev001", &deviceMutex)
	dtcontext.DGTwinList.Store("dev002", &dgTwin2)
	var deviceMutex2	sync.Mutex
	dtcontext.DGTwinMutex.Store("dev002", &deviceMutex2)

	
	newTwin := types.DigitalTwin{
		ID:	"dev001",
	}
	newTwin2 := types.DigitalTwin{
		ID:	"dev002",
	}

	twins := []types.DigitalTwin{newTwin, newTwin2}
	bytes, err := types.BuildTwinMessage(types.DGTWINS_OPS_TWINSGET, twins)
	if err == nil {
		modelMsg := dtcontext.BuildModelMessage("edge/app", types.MODULE_NAME, 
							types.DGTWINS_OPS_TWINSGET, types.DGTWINS_MODULE_TWINS, bytes)
		comm <- modelMsg
		heartBeat <- "ping"
	}

	go deviceModule.Start()
	time.Sleep(10 * time.Millisecond)

	v, ok := <-dtcontext.CommChan["comm"]
	if !ok {
		t.Errorf("channel closed")
	}
	message, isMsgType := v.(*model.Message )
	if !isMsgType {
		t.Errorf("Not message type")
	}

	content, ok := message.Content.([]byte)
	if !ok {
		t.Errorf("invaliad message content")
	}
	var dgTwinMsg types.DGTwinMessage
	json.Unmarshal(content, &dgTwinMsg)
	
	t.Logf("dgTwinMsg (%v)", dgTwinMsg)
	for _, dgTwin := range dgTwinMsg.Twins	{
		deviceID := dgTwin.ID
		if deviceID != "dev001" && deviceID != "dev002" {
			t.Errorf("deviceID != dev001 && deviceID != dev002")
		} 
	}

	heartBeat <- "stop"
}

