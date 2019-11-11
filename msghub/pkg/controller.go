package msghub

import (
	"k8s.io/klog"
	"github.com/jwzl/beehive/pkg/core/context"
)

type Controller struct {
	EdgeID		string
	context		*context.Context
	stopChan   chan struct{}
}
