package event

import "kcl-lang.io/kpm/pkg/client"

// KpmEvent handles diagnostic events.
type KpmEvent struct {
	client *client.KpmClient
}

// NewKpmEvent creates a new KpmEvent.
func NewKpmEvent(client *client.KpmClient) *KpmEvent {
	return &KpmEvent{client: client}
}

// Log writes a diagnostic message.
func (e *KpmEvent) Log(message string) {
	e.client.WriteLog(message)
}
