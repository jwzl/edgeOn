package dtmodule

import (
	"time"
	"testing"
	"github.com/jwzl/beehive/pkg/core/context"
	"github.com/jwzl/edgeOn/digitaltwin/pkg/types"
	"github.com/jwzl/edgeOn/digitaltwin/pkg/dtcontext"
)

func TestNewCommModule(t *testing.T) {
	commModule := NewCommModule()
	if commModule == nil {
		t.Fatal("Failed to create comm module.")
	}
}

func TestStart(t *testing.T) {
	ctx := context.GetContext(context.MsgCtxTypeChannel)
	dtcontext := dtcontext.NewDTContext(ctx)
	commModule := NewCommModule()
	comm := make(chan interface{}, 128)
	heartBeat := make(chan interface{}, 128)
	
	commModule.InitModule(dtcontext, comm, heartBeat, nil) 

	modelMsg := dtcontext.BuildModelMessage(types.MODULE_NAME, "device", 
					types.DGTWINS_OPS_GET, types.DGTWINS_RESOURCE_DEVICE, "helloworld") 
	modelMsg.Header.ID ="message"

	//send message
	comm <- modelMsg

	t.Run("comm_test", func(t *testing.T){
		go commModule.Start()
		time.Sleep(1 * time.Millisecond)	

		_, exist := commModule.context.MessageCache.Load("message")
		if !exist {
			t.Errorf("no message recieved!")
		}
		heartBeat <- "ping"
	})

	t.Log("recieve message")
	heartBeat <- "stop"
}
