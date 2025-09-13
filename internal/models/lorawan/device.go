package lorawan

// LoRaWANDevice represents a LoRaWAN device
type LoRaWANDevice struct {
	DevEUI     string `json:"dev_eui"`
	DeviceName string `json:"device_name"`
}

// GetDeviceID returns the DevEUI as primary identifier
func (d *LoRaWANDevice) GetDeviceID() string {
	return d.DevEUI
}

// GetDeviceName returns the device name
func (d *LoRaWANDevice) GetDeviceName() string {
	return d.DeviceName
}
