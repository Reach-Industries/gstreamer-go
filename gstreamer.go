package gstreamer

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-base-1.0 gstreamer-app-1.0 gstreamer-plugins-base-1.0 gstreamer-video-1.0 gstreamer-audio-1.0 gstreamer-plugins-bad-1.0
#include "gstreamer.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"sync"
	"unsafe"
)

func init() {
	C.gstreamer_init()
}

// StartGlibMainThreadLoop starts GLib's main loop
// It needs to be called from the process' main thread
// Because many gstreamer plugins require access to the main thread
// See: https://golang.org/pkg/runtime/#LockOSThread
func StartGlibMainThreadLoop() {
	C.gstreamer_receive_start_mainloop()
}

var pipelines = make(map[int]*Pipeline)
var elements = make(map[int]*Element)
var gstreamerLock sync.Mutex
var gstreamerIdGenerate = 10000

func New(pipelineStr string) (*Pipeline, error) {
	pipelineStrUnsafe := C.CString(pipelineStr)
	defer C.free(unsafe.Pointer(pipelineStrUnsafe))
	cpipeline := C.gstreamer_create_pipeline(pipelineStrUnsafe)
	if cpipeline == nil {
		return nil, errors.New("create pipeline error")
	}

	pipeline := &Pipeline{
		pipeline: cpipeline,
	}

	gstreamerLock.Lock()
	defer gstreamerLock.Unlock()
	gstreamerIdGenerate += 1
	pipeline.id = gstreamerIdGenerate
	pipelines[pipeline.id] = pipeline
	return pipeline, nil
}

func (p *Pipeline) PullMessage() <-chan *Message {
	p.messages = make(chan *Message, 5)
	C.gstreamer_pipeline_but_watch(p.pipeline, C.int(p.id))
	return p.messages
}

func (p *Pipeline) Start() {
	C.gstreamer_pipeline_start(p.pipeline, C.int(p.id))
}

func (p *Pipeline) Pause() {
	C.gstreamer_pipeline_pause(p.pipeline)
}

// Stops and disposes of the pipeline
func (p *Pipeline) Stop() {
	gstreamerLock.Lock()
	delete(pipelines, p.id)
	gstreamerLock.Unlock()
	if p.messages != nil {
		close(p.messages)
	}
	C.gstreamer_pipeline_stop(p.pipeline)
	C.gstreamer_pipeline_unref(p.pipeline)
}

func (p *Pipeline) SendEOS() {
	C.gstreamer_pipeline_sendeos(p.pipeline)
}

func (p *Pipeline) SetAutoFlushBus(flush bool) {
	gflush := gbool(flush)
	C.gstreamer_pipeline_set_auto_flush_bus(p.pipeline, gflush)
}

func (p *Pipeline) GetAutoFlushBus() bool {
	gflush := C.gstreamer_pipeline_get_auto_flush_bus(p.pipeline)
	return gobool(gflush)
}

func (p *Pipeline) GetDelay() uint64 {

	delay := C.gstreamer_pipeline_get_delay(p.pipeline)
	return uint64(delay)
}

func (p *Pipeline) SetDelay(delay uint64) {
	C.gstreamer_pipeline_set_delay(p.pipeline, C.guint64(delay))
}

func (p *Pipeline) GetLatency() uint64 {

	latency := C.gstreamer_pipeline_get_latency(p.pipeline)
	return uint64(latency)
}

func (p *Pipeline) SetLatency(latency uint64) {
	C.gstreamer_pipeline_set_latency(p.pipeline, C.guint64(latency))
}

func (p *Pipeline) FindElement(name string) *Element {
	elementName := C.CString(name)
	defer C.free(unsafe.Pointer(elementName))
	gelement := C.gstreamer_pipeline_findelement(p.pipeline, elementName)
	if gelement == nil {
		return nil
	}
	element := &Element{
		element: gelement,
	}

	gstreamerLock.Lock()
	defer gstreamerLock.Unlock()
	gstreamerIdGenerate += 1
	element.id = gstreamerIdGenerate
	elements[element.id] = element

	return element
}

func (e *Element) SetCap(cap string) {
	capStr := C.CString(cap)
	defer C.free(unsafe.Pointer(capStr))
	C.gstreamer_set_caps(e.element, capStr)
}

func (e *Element) Push(buffer []byte) {

	b := C.CBytes(buffer)
	defer C.free(unsafe.Pointer(b))
	C.gstreamer_element_push_buffer(e.element, b, C.int(len(buffer)))
}

func (e *Element) Poll() <-chan []byte {
	if e.out == nil {
		e.out = make(chan []byte, 10)
		C.gstreamer_element_pull_buffer(e.element, C.int(e.id))
	}
	return e.out
}

func (e *Element) Stop() {
	gstreamerLock.Lock()
	delete(elements, e.id)
	gstreamerLock.Unlock()
	if e.stop {
		return
	}
	if e.out != nil {
		e.stop = true
		close(e.out)
	}

}

//export goHandleSinkBuffer
func goHandleSinkBuffer(buffer unsafe.Pointer, bufferLen C.int, elementID C.int) {
	gstreamerLock.Lock()
	defer gstreamerLock.Unlock()
	if element, ok := elements[int(elementID)]; ok {
		if element.out != nil && !element.stop {
			element.out <- C.GoBytes(buffer, bufferLen)
		}
	} else {
		fmt.Printf("discarding buffer, no element with id %d", int(elementID))
	}
	C.free(buffer)
}

//export goHandleSinkEOS
func goHandleSinkEOS(elementID C.int) {
	gstreamerLock.Lock()
	defer gstreamerLock.Unlock()
	if element, ok := elements[int(elementID)]; ok {
		if element.out != nil && !element.stop {
			element.stop = true
			close(element.out)
		}
	}
}

//export goHandleBusMessage
func goHandleBusMessage(message *C.GstMessage, pipelineId C.int) {
	gstreamerLock.Lock()
	defer gstreamerLock.Unlock()
	msg := &Message{GstMessage: message}
	if pipeline, ok := pipelines[int(pipelineId)]; ok {
		if pipeline.messages != nil {
			pipeline.messages <- msg
		}
	} else {
		fmt.Printf("discarding message, no pipeline with id %d", int(pipelineId))
	}

}

// ScanPathForPlugins : Scans a given path for any gstreamer plugins and adds them to
// the gst_registry
func ScanPathForPlugins(directory string) {
	C.gst_registry_scan_path(C.gst_registry_get(), C.CString(directory))
}

func CheckPlugins(plugins []string) error {
	var plugin *C.GstPlugin
	registry := C.gst_registry_get()

	for _, pluginstr := range plugins {
		plugincstr := C.CString(pluginstr)
		plugin = C.gst_registry_find_plugin(registry, plugincstr)
		C.free(unsafe.Pointer(plugincstr))
		if plugin == nil {
			return fmt.Errorf("Required gstreamer plugin %s not found", pluginstr)
		}
	}

	return nil
}
