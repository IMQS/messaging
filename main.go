package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/IMQS/messaging/messaging"
	"github.com/julienschmidt/httprouter"
)

type configuration struct {
	HTTPPort int
}

// main starts a new http server and listens requests on a number of routes.
// It calls the messaging package to handle these requests.
func main() {
	// Read config file
	conf := configuration{}
	file, err := os.Open("messaging/conf.json")
	if err != nil {
		log.Fatal("Open:", err)
		return
	}

	decoder := json.NewDecoder(file)

	if err = decoder.Decode(&conf); err != nil {
		log.Println("Error parsing config file:", err)
		return
	}
	file.Close()

	// Perform initial configuration
	err = messaging.Bootstrap()
	if err != nil {
		log.Println(err)
		return
	}

	// Initialize routes
	address := fmt.Sprintf("%v:%v", "localhost", conf.HTTPPort)
	router := httprouter.New()
	router.GET("/messageStatus/:msisdn", messaging.HandleMessageStatus)
	router.POST("/sendSMS", messaging.HandleSendSMS)
	router.POST("/normalize", messaging.HandleNormalize)

	log.Printf("Messaging is listening on %v", address)
	err = http.ListenAndServe(address, router)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
		return
	}
}
