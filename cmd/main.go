package main

import (
	"fmt"

	"github.com/IMQS/cli"
	"github.com/IMQS/messaging"
)

func main() {
	app := cli.App{}
	app.Description = "messaging -c=configfile [options] command"
	app.DefaultExec = exec
	app.AddCommand("run", "Run the messaging service")
	app.AddValueOption("c", "configfile", "Configuration file. This option is mandatory")
	app.Run()
}

func exec(cmdName string, args []string, options cli.OptionSet) int {
	configFile := options["c"]
	if configFile == "" {
		fmt.Printf("You must specify a config file\n")
		return 1
	}

	server := messaging.MessagingServer{}
	err := server.Config.NewConfig(configFile)
	if err != nil {
		fmt.Printf("Error loading messaging config: %v\n", err)
		return 1
	}

	if err := server.Initialize(); err != nil {
		fmt.Printf("Error initializing messaging server: %v\n", err)
		return 1
	}

	run := func() {
		err := server.StartServer()
		if err != nil {
			server.Log.Errorf("%v\n", err)
			return
		}
	}

	switch cmdName {
	case "run":
		if !messaging.RunAsService(run) {
			run()
		}
	default:
		fmt.Printf("Unknown command %v\n", cmdName)
	}

	return 0
}
