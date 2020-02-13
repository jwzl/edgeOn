package cmd

import(
	"k8s.io/klog"
	"github.com/spf13/cobra"
	"github.com/jwzl/beehive/pkg/core"
	"github.com/jwzl/edgeOn/msghub"
	"github.com/jwzl/edgeOn/dgtwin"
	"github.com/jwzl/edgeOn/eventbus"
)

/*
* new app command
*/
func NewAppCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use: "edgeOn",
		Long: `edgeOn is an open source appliaction framework for edge computing. 
		It's the implementation of the digital twin on edge. each physical device 
		has a digital twin to describe this device on edge, and operate this twin 
		is equal to do this physical device. these twins can be created/updated/
		deleted/Get/Watch by cloud app or edge app. edgeOn has three parts: cloud 
		part, edge part and device part. edgeOn is either deployed on container 
		environment or non-container environment.. `,
		Run: func(cmd *cobra.Command, args []string) {
			//TODO: To help debugging, immediately log version
			klog.Infof("###########  Start the edgeOn client...! ###########")
			registerModules()
			// start all modules
			core.Run()
		},
	}

	return cmd
}

// register all module into beehive.
func registerModules(){
	dgtwin.Register()
	msghub.Register()
	eventbus.Register()
}
