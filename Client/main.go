package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const Version = "v1"
const ServerAddress = "192.168.1.183"
const TargetPort = "8888"
const HeartbeatInterval = 0.5
const ShutdownListenerPort = "9000"

const ClientStartupType = 0
const ClientHeartbeatType = 1
const ClientShutdownStartType = 2
const ClientShutdownEndType = 3

type ClientRequest struct {
	Hostname string `json:"hostname"`
	Type int `json:"type"`
	Version string `json:"version"`
}

var active bool
var shuttingDown bool

func main() {
	active = true
	shuttingDown = false
	sendMessageToServer(ClientStartupType)
	log.Printf("Sent startup code to server")
	log.Printf("Starting heartbeat loop")
	go heartbeatLoop()
	http.HandleFunc("/", shutdownHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", ShutdownListenerPort), nil))
}

func heartbeatLoop() {
	for active {
		sendMessageToServer(ClientHeartbeatType)
		log.Printf("Sending heartbeat")
		time.Sleep(HeartbeatInterval * 1000 * time.Millisecond)
	}
	for shuttingDown {

	}
	time.Sleep(1 * time.Second)
	os.Exit(0)
}

func sendMessageToServer(code int) {
	server := fmt.Sprintf("http://%v:%v", ServerAddress, TargetPort)
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Failed to get server hostname: %v", err)
	}
	req := ClientRequest{
		Hostname: hostname,
		Type:     code,
		Version:  Version,
	}
	reqBuffer := marshalRequestToByteBuffer(req)
	_, err = http.Post(server, "application/json", reqBuffer)
	if err != nil {
		log.Fatalf("Failed to send request: %v", err)
	}
}

func marshalRequestToByteBuffer(request ClientRequest) *bytes.Buffer {
	out, err := json.Marshal(request)
	if err != nil {
		log.Fatalf("Error marshalling request to JSON: %v", err)
	}
	return bytes.NewBuffer(out)
}

func shutdownHandler(w http.ResponseWriter, r *http.Request) {
	sendMessageToServer(ClientShutdownStartType)
	shuttingDown = true
	active = false
	time.Sleep(5 * time.Second)
	sendMessageToServer(ClientShutdownEndType)
	fmt.Fprintf(w, "Successfully Shutdown Client")
	shuttingDown = false
}
