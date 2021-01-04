package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/log"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	z2m        = "zigbee2mqtt"
	mqttServer = "tcp://127.0.0.1:1883"
	pin        = "11223399"
)

var (
	mqttClient      mqtt.Client
	bridgeOnline    bool
	devices         []*Device
	bridgeAccessory *accessory.Bridge
	transport       hc.Transport
)

type topicListener struct {
	Topic   string
	Receive func(tl *topicListener, client mqtt.Client, message mqtt.Message)
}

var topics = []topicListener{
	{Topic: z2m + "/bridge/state", Receive: func(tl *topicListener, client mqtt.Client, message mqtt.Message) {
		state := message.Payload()
		log.Info.Printf("Bridge: %s", state)
		if bytes.Equal(state, []byte("online")) {
			bridgeOnline = true
		} else {
			bridgeOnline = false
		}
	}},
	{Topic: z2m + "/bridge/info", Receive: func(tl *topicListener, client mqtt.Client, message mqtt.Message) {
		var config Z2MConfig
		err := json.Unmarshal(message.Payload(), &config)
		if err != nil {
			log.Info.Printf("Unmarshal %v", err)
			return
		}
		//spew.Dump(config)
	}},
	{Topic: z2m + "/bridge/devices", Receive: func(tl *topicListener, client mqtt.Client, message mqtt.Message) {
		//log.Printf("Devices: %s", message.Payload())
		var zigbeeDevices []*Z2MDevice
		err := json.Unmarshal(message.Payload(), &zigbeeDevices)
		if err != nil {
			log.Info.Printf("Unmarshal %v", err)
			return
		}
		var ellegibleDevices []*Device
		for _, z := range zigbeeDevices {
			// Skip over ourselves
			if z.Type == Z2MDeviceTypeCoordinator {
				continue
			}
			// Skip over incomplete or unsupported devices
			if !z.Interviewing && z.InterviewCompleted && z.Supported {
				d := newDevice(z)
				if d != nil {
					ellegibleDevices = append(ellegibleDevices, d)
				} else {
					log.Info.Printf("skipping unsupported %v (%v, %v, %v)", z.FriendlyName, z.Definition.Model, z.Definition.Description, z.IeeeAddress)
				}
			}
		}
		//spew.Dump(ellegibleDevices)
		if transport != nil {
			<-transport.Stop()
			transport = nil
		}
		for _, d := range devices {
			var topics []string
			if d.Subscribed {
				topics = append(topics, z2m+"/"+d.FriendlyName)
			}
			if len(topics) > 0 {
				token := mqttClient.Unsubscribe(topics...)
				token.Wait()
				if token.Error() != nil {
					log.Info.Printf("Unsubscribe: %v", token.Error())
				}
			}
		}
		devices = ellegibleDevices
		accessories := make([]*accessory.Accessory, len(devices))
		for i, d := range devices {
			accessories[i] = d.Accessory
		}
		transport, err = hc.NewIPTransport(hc.Config{Pin: pin, StoragePath: "./.db"}, bridgeAccessory.Accessory, accessories...)
		if err != nil {
			log.Info.Printf("NewIPTransport: %s", err)
		}
		go func() {
			transport.Start()
		}()
	}},
}

func (tl *topicListener) onMessageReceived(client mqtt.Client, message mqtt.Message) {
	tl.Receive(tl, client, message)
}

func main() {
	log.Debug.Enable()
	bridgeAccessory = accessory.NewBridge(accessory.Info{
		Name:         "Casa Terraza Zigbee",
		Model:        "hkz2m",
		Manufacturer: "Jens-Uwe Mager",
	})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	timer := time.NewTimer(1 * time.Second)
	connected := false
	connOpts := mqtt.NewClientOptions().
		AddBroker(mqttServer).
		SetClientID("hkz2m")

	connOpts.OnConnect = func(c mqtt.Client) {
		for i := range topics {
			if token := c.Subscribe(topics[i].Topic, 0, topics[i].onMessageReceived); token.Wait() && token.Error() != nil {
				log.Info.Fatal(token.Error())
			}
		}
	}
	connOpts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Info.Printf("connection lost: %v", err)
		connected = false
		timer.Reset(1 * time.Second)
	}

	mqttClient = mqtt.NewClient(connOpts)
conLoop:
	for {
		if !connected {
			token := mqttClient.Connect()
			_ = token.Wait() // can only return true ?!?
			if err := token.Error(); err != nil {
				log.Info.Printf("%s: %v", mqttServer, err)
			} else {
				log.Info.Printf("Connected to %s", mqttServer)
				connected = true
				timer.Stop()
			}
		}
		select {
		case <-c:
			break conLoop
		case t := <-timer.C:
			log.Info.Printf("timer: %v", t)
			timer.Reset(5 * time.Second)
		}
	}
	if transport != nil {
		<-transport.Stop()
		transport = nil
	}
	log.Info.Printf("Stopped")
}
