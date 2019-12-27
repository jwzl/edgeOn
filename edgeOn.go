package main

import (
	"os"
	"k8s.io/component-base/logs"
	"github.com/jwzl/edgeOn/cmd"
)


func main() {
	command := cmd.NewAppCommand()
	logs.InitLogs()
	defer logs.FlushLogs()	

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
