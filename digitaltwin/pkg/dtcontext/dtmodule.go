package dtcontext

type DTModule interface {
	Name() string
	//Init the digital twim module 		
	InitModule(dtc *DTContext, comm, heartBeat, confirm chan interface{})	
	// Start the module.
	Start()
}
