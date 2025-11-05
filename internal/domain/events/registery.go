package events

import (
	"fmt"
	"reflect"
)

var eventRegistry = make(map[EventType]reflect.Type)

func RegisterEvent(eventType EventType, event interface{}) {
	eventRegistry[eventType] = reflect.TypeOf(event)
}

func CreateEvent(eventType EventType) (interface{}, error) {
	t, ok := eventRegistry[eventType]
	if !ok {
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}
	return reflect.New(t).Interface(), nil
}

func init() {
	RegisterEvent(EventTypeImageProcessingRequested, ImageProcessingRequestedEvent{})
	RegisterEvent(EventTypeImageProcessingCompleted, ImageProcessingCompletedEvent{})
	RegisterEvent(EventTypeImageProcessingFailed, ImageProcessingFailedEvent{})
}
