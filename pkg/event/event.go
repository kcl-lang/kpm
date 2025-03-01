package event

import "kcl-lang.io/kpm/pkg/client"

type KpmEvent struct {
	client *client.KpmClient
}

func NewKpmEvent(client *client.KpmClient) *KpmEvent {
	return &KpmEvent{client: client}
}

func (e *KpmEvent) Log(message string) {
	e.client.WriteLog(message)
}
