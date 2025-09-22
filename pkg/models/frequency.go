package models

// FrequencyPoint represents a single frequency measurement
type FrequencyPoint struct {
	Frequency float64 `json:"frequency" doc:"Frequency in Hz"`
	Magnitude float64 `json:"magnitude" doc:"Magnitude in dB"`
}
