package dtmodule

import (
	"k8s.io/klog"
	"github.com/jwzl/edgeOn/dgtwin/types"
	"github.com/jwzl/edgeOn/dgtwin/dtcontext"
)

func NewDTModule(moduleName string) dtcontext.DTModule {

	switch moduleName {
	case types.DGTWINS_MODULE_COMM:
		return NewCommModule()
	case types.DGTWINS_MODULE_PROPERTY:
		return NewPropertyModule()
	case types.DGTWINS_MODULE_TWINS:
		return NewTwinModule()
	default:
		klog.Errorf("moduleName is invaild.")
		return nil
	}
}
