package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const Port = 8888
const LogFile = "server.log"
const HeartbeatTimeout = 2

const ClientStartupType = 0
const ClientHeartbeatType = 1
const ClientShutdownStartType = 2
const ClientShutdownEndType = 3


type ClientRequest struct {
	Hostname string `json:"hostname"`
	Type int `json:"type"`
	Version string `json:"version"`
}

type Client struct {
	Hostname string
	Version string
	LastHeartbeat time.Time
	Active bool
}

type SafeClientStore struct {
	clients map[string]*Client
	lock *sync.RWMutex
}

var clientStore *SafeClientStore

func main() {
	f, err := os.OpenFile(LogFile, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}
	defer f.Close()
	mw := io.MultiWriter(f, os.Stdout)
	log.SetOutput(mw)
	log.Printf("Starting Server")
	clientStore = &SafeClientStore{
		clients: map[string]*Client{},
		lock:    new(sync.RWMutex),
	}
	go asyncHeartbeatMonitor()
	http.HandleFunc("/", handleClientRequest)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", Port), nil))
}

func handleClientRequest(w http.ResponseWriter, r *http.Request) {
	var req ClientRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Fatalf("Failed to to parse request : %v\n", err)
	}
	switch req.Type {
	case ClientStartupType:
		registerNewClient(req.Hostname, req.Version)
	case ClientHeartbeatType:
		heartbeatClient(req.Hostname, req.Version)
	case ClientShutdownStartType:
		startShutdownClient(req.Hostname, req.Version)
	case ClientShutdownEndType:
		endShutdownClient(req.Hostname, req.Version)
	default:
		break
	}
}

func registerNewClient(hostname string, version string) {
	client := Client{
		Hostname:      hostname,
		Version:       version,
		LastHeartbeat: time.Now(),
		Active: true,
	}
	clientStore.lock.Lock()
	defer clientStore.lock.Unlock()
	// check if client is already in store
	if existingClient, exists := clientStore.clients[client.Hostname]; exists {
		log.Printf("Client %v attempted to register with version %v but already exists with version %v",
			client.Hostname, client.Version, existingClient.Version)
		return
	}
	clientStore.clients[client.Hostname] = &client
	log.Printf("Registered %v at version %v", client.Hostname, client.Version)
	return
}

func getExistingClient(hostname string, version string) *Client {
	// check if client is already in store
	var existingClient *Client
	var exists bool
	if existingClient, exists = clientStore.clients[hostname]; !exists {
		log.Printf("Recieved request from %v but it doesnt exist in store",
			hostname)
		return nil
	}
	if existingClient.Version != version {
		log.Printf("Version mismatch from %v: registered version: %v, recieved version: %v",
			hostname, existingClient.Version, version)
	}
	return existingClient
}

func heartbeatClient(hostname string, version string) {
	clientStore.lock.Lock()
	defer clientStore.lock.Unlock()
	existingClient := getExistingClient(hostname, version)
	if existingClient == nil {
		return
	}
	existingClient.LastHeartbeat = time.Now()
}

func startShutdownClient(hostname string, version string) {
	clientStore.lock.Lock()
	defer clientStore.lock.Unlock()
	existingClient := getExistingClient(hostname, version)
	if existingClient == nil {
		return
	}
	log.Printf("Recieved shutdown signal from %v (%v)", hostname, version)
	existingClient.Active = false
}

func endShutdownClient(hostname string, version string) {
	clientStore.lock.Lock()
	defer clientStore.lock.Unlock()
	existingClient := getExistingClient(hostname, version)
	if existingClient == nil {
		return
	}
	log.Printf("%v (%v) finished shutdown - removing", hostname, version)
	delete(clientStore.clients, hostname)
}

func asyncHeartbeatMonitor() {
	for {
		clientStore.lock.RLock()
		now := time.Now()
		for _, client := range clientStore.clients {
			if !client.Active {
				continue
			}
			if client.LastHeartbeat.Add(time.Second).Before(now) {
				log.Printf("Havent received heartbeat from %v (%v) in %v seconds",
					client.Hostname, client.Version, HeartbeatTimeout)
				delete(clientStore.clients, client.Hostname)
			}
		}
		clientStore.lock.RUnlock()
		time.Sleep(50 * time.Millisecond)
	}
}
