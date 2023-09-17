package config

// Event Constant
const (
	UpdateType = "UPDATE"
	DeleteType = "DELETE"
	CreateType = "CREATE"
)

type Event struct {
	EventSource string
	EventType   string
	Key         string
	Value       string
	HasUpdated  bool
}

func newEvent(eventSource, eventType string, key string, value string) *Event {
	return &Event{
		EventSource: eventSource,
		EventType:   eventType,
		Key:         key,
		Value:       value,
		HasUpdated:  false,
	}

}

func PopulateEvents(source string, currentConfig, updatedConfig map[string]string) ([]*Event, error) {
	events := make([]*Event, 0)

	// generator create and update event
	for key, value := range updatedConfig {
		currentValue, ok := currentConfig[key]
		if !ok { // if new configuration introduced
			events = append(events, newEvent(source, CreateType, key, value))
		} else if currentValue != value {
			events = append(events, newEvent(source, UpdateType, key, value))
		}
	}

	// generator delete event
	for key, value := range currentConfig {
		_, ok := updatedConfig[key]
		if !ok { // when old config not present in new config
			events = append(events, newEvent(source, DeleteType, key, value))
		}
	}
	return events, nil
}
