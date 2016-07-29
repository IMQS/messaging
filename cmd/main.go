package main

import (
	"fmt"
	"log"

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

	err := messaging.NewConfig(configFile)
	if err != nil {
		fmt.Printf("Error constructing messaging config: %v", err)
		// CR: Should return 1
	}

	run := func() {
		err := messaging.StartServer()
		if err != nil {
			log.Fatal(err)
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
		// CR: Should return 1
	}

	return 0
}
