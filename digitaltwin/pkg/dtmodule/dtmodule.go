package dtmodule

import (
	"github.com/jwzl/edgeOn/digitaltwin/pkg/dtcontext"
)

type dtmodule interface {
	Name() string
	//Init the digital twim module 		
	Init_Module(dtc *dtcontext.DTContext, comm, heartBeat, confirm chan interface{})	
	// Start the module.
	Start()
}
