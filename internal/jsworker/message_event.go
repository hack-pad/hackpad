package jsworker

import (
	"fmt"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/common"
)

type MessageEvent struct {
	Data   js.Value
	Target *MessagePort
}

func parseMessageEvent(v js.Value) (_ MessageEvent, err error) {
	defer common.CatchException(&err)
	target, err := WrapMessagePort(v.Get("target"))
	return MessageEvent{
		Data:   v.Get("data"),
		Target: target,
	}, err
}

type MessageEventErr struct {
	MessageEvent
}

func (m MessageEventErr) Error() string {
	return fmt.Sprintf("Failed to deserialize message: %+v", m.MessageEvent)
}
