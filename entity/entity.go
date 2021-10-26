package entity


type CommandTemperature struct {
	SerialNumber int `json:"serial_number" validate:"required"`
	Temperature  float32 `json:"temperature" validate:"required"`
	Zone         int `json:"zone" validate:"required"`
}

type CommandTemperatureBySensor struct {
	SerialNumber int `json:"serial_number" validate:"required"`
	Type  int `json:"type" validate:"required"`
	Zone         int `json:"zone" validate:"required"`
}

type Response struct {
	Status int `json:"status"`
	Message string `json:"message,omitempty"`
	Token string `json:"token,omitempty"`
	Data interface{} `json:"data,omitempty"`
}