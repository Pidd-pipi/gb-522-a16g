package constants

type EventType string

const (
	EventConnector EventType = "connector"
	EventSplice    EventType = "splice"
	EventBend      EventType = "bend"
	EventBreak     EventType = "break"
	EventEnd       EventType = "end"
	EventUnknown   EventType = "unknown"
)

var eventTypes = map[EventType]struct{}{
	EventConnector: {}, EventSplice: {}, EventBend: {}, EventBreak: {}, EventEnd: {}, EventUnknown: {},
}

func (v EventType) Valid() bool {
	_, ok := eventTypes[v]
	return ok
}

func EventTypes() []EventType {
	return []EventType{EventConnector, EventSplice, EventBend, EventBreak, EventEnd, EventUnknown}
}
