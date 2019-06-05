package sensibo

import "time"

type PodList struct {
	Status string `json:"status"`
	Pods   []Pod  `json:"result"`
}
type AcState struct {
	On                bool   `json:"on"`
	FanLevel          string `json:"fanLevel"`
	TemperatureUnit   string `json:"temperatureUnit"`
	TargetTemperature int    `json:"targetTemperature"`
	Mode              string `json:"mode"`
	Swing             string `json:"swing"`
}
type Time struct {
	SecondsAgo int       `json:"secondsAgo"`
	Time       time.Time `json:"time"`
}
type Measurements struct {
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
}
type TemperatureState struct {
	On                bool   `json:"on"`
	FanLevel          string `json:"fanLevel"`
	TemperatureUnit   string `json:"temperatureUnit"`
	TargetTemperature int    `json:"targetTemperature"`
	Mode              string `json:"mode"`
}
type SmartMode struct {
	DeviceUID                string           `json:"deviceUid"`
	HighTemperatureThreshold float64          `json:"highTemperatureThreshold"`
	Type                     string           `json:"type"`
	LowTemperatureState      TemperatureState `json:"lowTemperatureState"`
	Enabled                  bool             `json:"enabled"`
	HighTemperatureState     TemperatureState `json:"highTemperatureState"`
	LowTemperatureThreshold  float64          `json:"lowTemperatureThreshold"`
}
type Pod struct {
	AcState      AcState      `json:"acState"`
	Measurements Measurements `json:"measurements"`
	SmartMode    SmartMode    `json:"smartMode"`
	ID           string       `json:"id"`
}
