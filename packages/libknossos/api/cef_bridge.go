package main

//#include <stdlib.h>
//#include <stdint.h>
//#include <string.h>
//
//typedef void (*KnossosLogCallback)(uint8_t level, char* message, int length);
//typedef void (*KnossosMessageCallback)(void* message, int length);
//
//typedef struct {
//  const char* settings_path;
//  const char* resource_path;
//  int settings_len;
//  int resource_len;
//  KnossosLogCallback log_cb;
//  KnossosMessageCallback message_cb;
//} KnossosInitParams;
//
//typedef struct {
//	char* header_name;
//  char* value;
//  size_t header_len;
//  size_t value_len;
//} KnossosHeader;
//
//typedef struct {
//  KnossosHeader* headers;
//	void* response_data;
//  int status_code;
//  uint8_t header_count;
//  size_t response_length;
//} KnossosResponse;
//
//#ifndef GO_CGO_EXPORT_PROLOGUE_H
//static void call_log_cb(KnossosLogCallback cb, uint8_t level, char* message, int length) {
//	cb(level, message, length);
//}
//
//static void call_message_cb(KnossosMessageCallback cb, void* message, int length) {
//  cb(message, length);
//}
//
//static KnossosResponse* make_response() {
//  return (KnossosResponse*) malloc(sizeof(KnossosResponse));
//}
//
//static KnossosHeader* make_header_array(uint8_t length) {
//  return (KnossosHeader*) malloc(sizeof(KnossosHeader) * length);
//}
//
//static void set_header(KnossosHeader* harray, uint8_t idx, _GoString_ name, _GoString_ value) {
//  KnossosHeader* hdr = &harray[idx];
//  hdr->header_len = _GoStringLen(name);
//  hdr->header_name = (char*)malloc(hdr->header_len);
//  memcpy(hdr->header_name, _GoStringPtr(name), hdr->header_len);
//
//  hdr->value_len = _GoStringLen(value);
//  hdr->value = (char*)malloc(hdr->value_len);
//  memcpy(hdr->value, _GoStringPtr(value), hdr->value_len);
//}
//
//static void set_body(KnossosResponse* response, _GoString_ body) {
//  response->response_length = _GoStringLen(body);
//  response->response_data = (void*)malloc(response->response_length);
//  memcpy(response->response_data, _GoStringPtr(body), response->response_length);
//}
//
//void KnossosFreeKnossosResponse(KnossosResponse* response) {
//  for (int i = 0; i < response->header_count; i++) {
//    KnossosHeader *hdr = &response->headers[i];
//    free(hdr->header_name);
//    free(hdr->value);
//  }
//  free(response->headers);
//  free(response->response_data);
//  free(response);
//}
//#else
//extern void KnossosFreeKnossosResponse(KnossosResponse* response);
//#endif
//
//#define KNOSSOS_LOG_INFO 1
//#define KNOSSOS_LOG_WARNING 2
//#define KNOSSOS_LOG_ERROR 3
//#define KNOSSOS_LOG_FATAL 4
import "C"

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"unsafe"

	"github.com/ngld/knossos/packages/api/client"
	"github.com/ngld/knossos/packages/libknossos/pkg/api"
	"github.com/ngld/knossos/packages/libknossos/pkg/twirp"
	"github.com/rotisserie/eris"
	"google.golang.org/protobuf/proto"
)

var (
	ready       = false
	logLevelMap = map[api.LogLevel]C.uint8_t{
		api.LogInfo:  C.KNOSSOS_LOG_INFO,
		api.LogWarn:  C.KNOSSOS_LOG_WARNING,
		api.LogError: C.KNOSSOS_LOG_ERROR,
		api.LogFatal: C.KNOSSOS_LOG_FATAL,
	}

	staticRoot   string
	settingsPath string
	logCb        C.KnossosLogCallback
	messageCb    C.KnossosMessageCallback
	server       http.Handler
)

func Log(level api.LogLevel, msg string, args ...interface{}) {
	finalMsg := fmt.Sprintf(msg, args...)
	cMsg := C.CString(finalMsg)

	C.call_log_cb(logCb, logLevelMap[level], cMsg, C.int(len(finalMsg)))
	C.free(unsafe.Pointer(cMsg))
}

// KnossosInit has to be called exactly once before calling any other exported function.
//export KnossosInit
func KnossosInit(params *C.KnossosInitParams) bool {
	staticRoot = C.GoStringN(params.resource_path, params.resource_len)
	settingsPath = C.GoStringN(params.settings_path, params.settings_len)
	logCb = params.log_cb
	messageCb = params.message_cb
	ready = true

	var err error
	server, err = twirp.NewServer()
	if err != nil {
		Log(api.LogError, "Failed to init twirp: %+v", err)
		return false
	}

	return true
}

// KnossosHandleRequest handles an incoming request from CEF
//export KnossosHandleRequest
func KnossosHandleRequest(urlPtr *C.char, urlLen C.int, bodyPtr unsafe.Pointer, bodyLen C.int) *C.KnossosResponse {
	body := C.GoBytes(bodyPtr, bodyLen)
	reqURL := C.GoStringN(urlPtr, urlLen)

	ctx, cancel := context.WithCancel(context.Background())

	ctx = api.WithKnossosContext(ctx, api.KnossosCtxParams{
		SettingsPath:    settingsPath,
		ResourcePath:    staticRoot,
		LogCallback:     Log,
		MessageCallback: DispatchMessage,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(body))
	req.Header["Content-Type"] = []string{"application/protobuf"}

	if err == nil {
		twirpResp := newMemoryResponse()
		server.ServeHTTP(twirpResp, req)
		// Cancel any background operation still attached to the request context
		cancel()

		resp := C.make_response()
		resp.status_code = C.int(twirpResp.statusCode)
		resp.header_count = C.uint8_t(len(twirpResp.headers))
		resp.headers = C.make_header_array(resp.header_count)

		idx := C.uint8_t(0)
		for k, v := range twirpResp.headers {
			C.set_header(resp.headers, idx, k, strings.Join(v, ", "))
			idx++
		}

		body := twirpResp.resp.Bytes()
		resp.response_data = C.CBytes(body)
		resp.response_length = C.size_t(len(body))

		return resp
	}

	// Cleanup the unused context
	cancel()

	resp := C.make_response()
	resp.status_code = 503
	resp.header_count = 1
	resp.headers = C.make_header_array(1)

	C.set_header(resp.headers, 0, "Content-Type", "text/plain")
	C.set_body(resp, fmt.Sprintf("Error: %+v", err))

	return resp
}

// DispatchMessage forwards the passed message to the hosting application
func DispatchMessage(event *client.ClientSentEvent) error {
	data, err := proto.Marshal(event)
	if err != nil {
		return eris.Wrap(err, "Failed to marshal event")
	}

	dataPtr := C.CBytes(data)
	C.call_message_cb(messageCb, dataPtr, C.int(len(data)))
	C.free(dataPtr)
	return nil
}

func main() {}
