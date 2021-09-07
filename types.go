package gstreamer

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-base-1.0 gstreamer-app-1.0 gstreamer-plugins-base-1.0 gstreamer-video-1.0 gstreamer-audio-1.0 gstreamer-plugins-bad-1.0
#include "gstreamer.h"
*/
import "C"
import "unsafe"

type MessageType int

const (
	MESSAGE_UNKNOWN       MessageType = C.GST_MESSAGE_UNKNOWN
	MESSAGE_EOS           MessageType = C.GST_MESSAGE_EOS
	MESSAGE_ERROR         MessageType = C.GST_MESSAGE_ERROR
	MESSAGE_WARNING       MessageType = C.GST_MESSAGE_WARNING
	MESSAGE_INFO          MessageType = C.GST_MESSAGE_INFO
	MESSAGE_TAG           MessageType = C.GST_MESSAGE_TAG
	MESSAGE_BUFFERING     MessageType = C.GST_MESSAGE_BUFFERING
	MESSAGE_STATE_CHANGED MessageType = C.GST_MESSAGE_STATE_CHANGED
	MESSAGE_ANY           MessageType = C.GST_MESSAGE_ANY
)

type Message struct {
	GstMessage *C.GstMessage
}

func (v *Message) GetType() MessageType {
	c := C.toGstMessageType(unsafe.Pointer(v.native()))
	return MessageType(c)
}

func (v *Message) native() *C.GstMessage {
	if v == nil {
		return nil
	}
	return v.GstMessage
}

func (v *Message) GetTimestamp() uint64 {
	c := C.messageTimestamp(unsafe.Pointer(v.native()))
	return uint64(c)
}

func (v *Message) GetTypeName() string {
	c := C.messageTypeName(unsafe.Pointer(v.native()))
	return C.GoString(c)
}

func gbool(b bool) C.gboolean {
	if b {
		return C.gboolean(1)
	}
	return C.gboolean(0)
}
func gobool(b C.gboolean) bool {
	return b != 0
}

type Element struct {
	element *C.GstElement
	out     chan []byte
	stop    bool
	id      int
}

type Pipeline struct {
	pipeline *C.GstPipeline
	messages chan *Message
	id       int
}
