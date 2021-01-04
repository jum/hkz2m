package main

import (
	"encoding/json"
	"time"
)

// Z2MConfig is the current Z2M configuration.
type Z2MConfig struct {
	Commit string `json:"commit"`
	Config struct {
		Advanced struct {
			AdapterConcurrent           interface{}   `json:"adapter_concurrent"`
			AdapterDelay                interface{}   `json:"adapter_delay"`
			AvailabilityBlacklist       []interface{} `json:"availability_blacklist"`
			AvailabilityBlocklist       []interface{} `json:"availability_blocklist"`
			AvailabilityPasslist        []interface{} `json:"availability_passlist"`
			AvailabilityTimeout         int           `json:"availability_timeout"`
			AvailabilityWhitelist       []interface{} `json:"availability_whitelist"`
			CacheState                  bool          `json:"cache_state"`
			CacheStatePersistent        bool          `json:"cache_state_persistent"`
			CacheStateSendOnStartup     bool          `json:"cache_state_send_on_startup"`
			Channel                     int           `json:"channel"`
			Elapsed                     bool          `json:"elapsed"`
			ExtPanID                    []int         `json:"ext_pan_id"`
			HomeassistantDiscoveryTopic string        `json:"homeassistant_discovery_topic"`
			HomeassistantLegacyTriggers bool          `json:"homeassistant_legacy_triggers"`
			HomeassistantStatusTopic    string        `json:"homeassistant_status_topic"`
			LastSeen                    string        `json:"last_seen"`
			LegacyAPI                   bool          `json:"legacy_api"`
			LogDirectory                string        `json:"log_directory"`
			LogFile                     string        `json:"log_file"`
			LogLevel                    string        `json:"log_level"`
			LogOutput                   []string      `json:"log_output"`
			LogRotation                 bool          `json:"log_rotation"`
			LogSyslog                   struct {
			} `json:"log_syslog"`
			PanID            int    `json:"pan_id"`
			Report           bool   `json:"report"`
			SoftResetTimeout int    `json:"soft_reset_timeout"`
			TimestampFormat  string `json:"timestamp_format"`
		} `json:"advanced"`
		Ban           []interface{} `json:"ban"`
		Blocklist     []interface{} `json:"blocklist"`
		DeviceOptions struct {
		} `json:"device_options"`
		Devices map[string]struct {
			FriendlyName string `json:"friendly_name"`
		} `json:"devices"`
		Experimental struct {
			NewAPI bool   `json:"new_api"`
			Output string `json:"output"`
		} `json:"experimental"`
		ExternalConverters []interface{} `json:"external_converters"`
		Frontend           struct {
			Port int `json:"port"`
		} `json:"frontend"`
		Groups map[int]struct {
			Devices      []string `json:"devices"`
			FriendlyName string   `json:"friendly_name"`
			Optimistic   bool     `json:"optimistic"`
			Retain       bool     `json:"retain"`
		} `json:"groups"`
		Homeassistant bool `json:"homeassistant"`
		MapOptions    struct {
			Graphviz struct {
				Colors struct {
					Fill struct {
						Coordinator string `json:"coordinator"`
						Enddevice   string `json:"enddevice"`
						Router      string `json:"router"`
					} `json:"fill"`
					Font struct {
						Coordinator string `json:"coordinator"`
						Enddevice   string `json:"enddevice"`
						Router      string `json:"router"`
					} `json:"font"`
					Line struct {
						Active   string `json:"active"`
						Inactive string `json:"inactive"`
					} `json:"line"`
				} `json:"colors"`
			} `json:"graphviz"`
		} `json:"map_options"`
		Mqtt struct {
			BaseTopic                string `json:"base_topic"`
			ClientID                 string `json:"client_id"`
			ForceDisableRetain       bool   `json:"force_disable_retain"`
			IncludeDeviceInformation bool   `json:"include_device_information"`
			Server                   string `json:"server"`
		} `json:"mqtt"`
		Passlist   []interface{} `json:"passlist"`
		PermitJoin bool          `json:"permit_join"`
		Serial     struct {
			DisableLed bool   `json:"disable_led"`
			Port       string `json:"port"`
		} `json:"serial"`
		Whitelist []interface{} `json:"whitelist"`
	} `json:"config"`
	Coordinator struct {
		Meta struct {
			Maintrel     int `json:"maintrel"`
			Majorrel     int `json:"majorrel"`
			Minorrel     int `json:"minorrel"`
			Product      int `json:"product"`
			Revision     int `json:"revision"`
			Transportrev int `json:"transportrev"`
		} `json:"meta"`
		Type string `json:"type"`
	} `json:"coordinator"`
	LogLevel string `json:"log_level"`
	Network  struct {
		Channel       int    `json:"channel"`
		ExtendedPanID string `json:"extended_pan_id"`
		PanID         int    `json:"pan_id"`
	} `json:"network"`
	PermitJoin bool   `json:"permit_join"`
	Version    string `json:"version"`
}

// UnixEpoch is a time value that is stored as int64 unix epoch in json.
type UnixEpoch time.Time

// UnmarshalJSON converts a seconds til epoch int into a time value.
func (e *UnixEpoch) UnmarshalJSON(b []byte) error {
	var secs int64
	err := json.Unmarshal(b, &secs)
	if err != nil {
		return err
	}
	*e = UnixEpoch(time.Unix(secs, 0))
	return nil
}

// GoString is used by fmt %#v, we like to see RFC3339 format here.
func (e UnixEpoch) GoString() string {
	return time.Now().Format(time.RFC3339)
}
