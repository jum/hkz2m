package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/log"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Z2MAccessMask is the access bit mask that a particular feature allows.
type Z2MAccessMask int

const (
	// Z2MAccessPublished denotes the feature is published state.
	Z2MAccessPublished = (1 << iota)
	// Z2MAccessSet the feature can be set with /set
	Z2MAccessSet = (1 << iota)
	// Z2MAccessGet the feture can be retrieved with /get
	Z2MAccessGet = (1 << iota)
)

func (m Z2MAccessMask) String() string {
	masks := []string{"Published", "Set", "Get"}
	var ret strings.Builder
	for i, flag := range masks {
		if m&(1<<i) != 0 {
			if ret.Len() > 0 {
				ret.WriteRune('|')
			}
			ret.WriteString(flag)
		}
	}
	return fmt.Sprintf("0b%03b", m) + " [" + ret.String() + "]"
}

// Z2MFeature describes the features exposed by a device. Note that this
// recursive for composite features.
type Z2MFeature struct {
	Type        string        `json:"type"`
	Name        string        `json:"name"`
	Access      Z2MAccessMask `json:"access"`
	Description string        `json:"description"`
	Property    string        `json:"property"`
	Unit        string        `json:"unit,omitempty"`
	ValueMax    int           `json:"value_max,omitempty"`
	ValueMin    int           `json:"value_min,omitempty"`
	Endpoint    int           `json:"endpoint,omitempty"`
	Values      []string      `json:"values,omitempty"`
	Features    []Z2MFeature  `json:"features,omitempty"`
}

/*
func (f *Z2MFeature) String() string {
	var s strings.Builder
	return s.String()
}
*/

// Z2MDefinition holds the Z2M device definition and the functions the device
// exposes.
type Z2MDefinition struct {
	Description string       `json:"description"`
	Model       string       `json:"model"`
	Vendor      string       `json:"vendor"`
	Exposes     []Z2MFeature `json:"exposes"`
}

// Z2MTarget is the target of a binding.
type Z2MTarget struct {
	ID          int    `json:"id"`
	Endpoint    int    `json:"endpoint"`
	IeeeAddress string `json:"ieee_address"`
	Type        string `json:"type"`
}

// Z2MClusters are the device ZigBee clusters being bound to.
type Z2MClusters struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

// Z2MBinding describes the binding of a cluster to some target.
type Z2MBinding struct {
	Cluster string    `json:"cluster"`
	Target  Z2MTarget `json:"target,omitempty"`
}

// Z2MEndpoint describes the Bindings and available clustors.
type Z2MEndpoint struct {
	Bindings []Z2MBinding `json:"bindings"`
	Clusters Z2MClusters  `json:"clusters"`
}

const (
	// Z2MDeviceTypeEndDevice is all ZigBee devices that do not route traffic
	Z2MDeviceTypeEndDevice = "EndDevice"
	// Z2MDeviceTypeRouter is end devices that routes traffic in the mesh
	Z2MDeviceTypeRouter = "Router"
	// Z2MDeviceTypeCoordinator is the device that coordinates the whole net
	Z2MDeviceTypeCoordinator = "Coordinator"
)

// Z2MDevice is the info returned from z2m in its devices message.
type Z2MDevice struct {
	Type               string              `json:"type"`
	FriendlyName       string              `json:"friendly_name"`
	IeeeAddress        string              `json:"ieee_address"`
	InterviewCompleted bool                `json:"interview_completed"`
	Interviewing       bool                `json:"interviewing"`
	Supported          bool                `json:"supported"`
	NetworkAddress     int                 `json:"network_address"`
	PowerSource        string              `json:"power_source,omitempty"`
	DateCode           string              `json:"date_code,omitempty"`
	SoftwareBuildID    string              `json:"software_build_id,omitempty"`
	Definition         Z2MDefinition       `json:"definition"`
	Endpoints          map[int]Z2MEndpoint `json:"endpoints,omitempty"`
}

// Device embeds a Z2MDevice and assorted transient information.
type Device struct {
	*accessory.Accessory
	*Z2MDevice
	Subscribed bool
}

// LightState is the JSON reported by z2m for light bulbs.
type LightState struct {
	State      string
	Brightness int
	ColorTemp  int `json:"color_temp"`
	Color      struct {
		X float64
		Y float64
	}
	LinkQuality int
}

func newDevice(z *Z2MDevice) *Device {
	f := findSpecificFeatureType(z.Definition.Exposes)
	if f == nil {
		log.Info.Printf("skipping %v, %v, %v", z.Definition.Model, z.Definition.Description, z.FriendlyName)
		return nil
	}
	id, err := strconv.ParseUint(z.IeeeAddress[2:], 16, 64)
	if err != nil {
		log.Info.Printf("cannot parse %v as unt16: %v", z.IeeeAddress, err)
	}
	switch f.Type {
	case "light":
		log.Debug.Printf("light feature: %#v", f.Features)
		d := &Device{
			Z2MDevice: z,
		}
		a := accessory.NewColoredLightbulb(accessory.Info{
			Name:             d.FriendlyName,
			Model:            d.Definition.Model,
			Manufacturer:     d.Definition.Vendor,
			SerialNumber:     d.IeeeAddress,
			FirmwareRevision: d.SoftwareBuildID,
			ID:               id,
		})
		d.Accessory = a.Accessory
		a.Lightbulb.On.OnValueRemoteUpdate(func(on bool) {
			log.Debug.Printf("%v on: %v", d.FriendlyName, on)
			payload := "OFF"
			if on {
				payload = "ON"
			}
			token := mqttClient.Publish(z2m+"/"+d.FriendlyName+"/"+"set/state", 0, false, payload)
			go func() {
				token.Wait()
				if token.Error() != nil {
					log.Debug.Printf("publish state %v: %v", d.FriendlyName, token.Error())
				}
			}()
		})
		a.Lightbulb.Brightness.OnValueRemoteUpdate(func(brightness int) {
			log.Debug.Printf("%v brightness: %v", d.FriendlyName, brightness)
			payload := fmt.Sprintf("%v", brightness)
			token := mqttClient.Publish(z2m+"/"+d.FriendlyName+"/"+"set/brightness", 0, false, payload)
			go func() {
				token.Wait()
				if token.Error() != nil {
					log.Info.Printf("publish state %v: %v", d.FriendlyName, token.Error())
				}
			}()
		})
		a.Lightbulb.Hue.OnValueRemoteUpdate(func(hue float64) {
			log.Debug.Printf("%v hue: %v", d.FriendlyName, hue)
		})
		a.Lightbulb.Saturation.OnValueRemoteUpdate(func(saturation float64) {
			log.Debug.Printf("%v saturation: %v", d.FriendlyName, saturation)
		})
		token := mqttClient.Subscribe(z2m+"/"+d.FriendlyName, 0, func(client mqtt.Client, message mqtt.Message) {
			var state LightState
			err := json.Unmarshal(message.Payload(), &state)
			if err != nil {
				log.Info.Printf("Unmarshal %v: %v", string(message.Payload()), err)
				return
			}
			log.Debug.Printf("%v got: %#v", d.FriendlyName, state)
			a.Lightbulb.On.SetValue(state.State == "ON")
			a.Lightbulb.Brightness.SetValue(state.Brightness)
		})
		go func() {
			token.Wait()
			if token.Error() != nil {
				log.Info.Printf("subscribe %v: %v", d.FriendlyName, token.Error())
			} else {
				d.Subscribed = true
			}
		}()
		return d
	default:
		log.Info.Printf("Unknown feature: %#v", f)
		return nil
	}
}

func findSpecificFeatureType(features []Z2MFeature) *Z2MFeature {
	for _, f := range features {
		if len(f.Features) > 0 {
			return &f
		}
	}
	return nil
}
