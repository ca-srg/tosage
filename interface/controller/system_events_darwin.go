//go:build darwin
// +build darwin

package controller

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework IOKit

#import <Foundation/Foundation.h>
#import <IOKit/IOKitLib.h>
#import <IOKit/pwr_mgt/IOPMLib.h>
#import <IOKit/IOMessage.h>

// Forward declaration
void systemEventCallback(int eventType);

typedef void (*SleepWakeCallback)(int eventType);

static io_connect_t rootPort;
static IONotificationPortRef notifyPort;
static io_object_t notifierObject;
static CFRunLoopRef runLoop;

static void systemPowerCallback(void *refCon, io_service_t service, natural_t messageType, void *messageArgument) {
    switch (messageType) {
        case kIOMessageSystemWillSleep:
            systemEventCallback(0); // 0 = sleep
            IOAllowPowerChange(rootPort, (long)messageArgument);
            break;
        case kIOMessageSystemHasPoweredOn:
            systemEventCallback(1); // 1 = wake
            break;
        default:
            break;
    }
}

static int registerSystemPowerNotifications() {
    rootPort = IORegisterForSystemPower(NULL, &notifyPort, systemPowerCallback, &notifierObject);
    if (rootPort == 0) {
        return -1;
    }

    runLoop = CFRunLoopGetCurrent();
    CFRunLoopAddSource(runLoop, IONotificationPortGetRunLoopSource(notifyPort), kCFRunLoopDefaultMode);
    return 0;
}

static void runSystemEventLoop() {
    CFRunLoopRun();
}

static void stopSystemEventLoop() {
    if (runLoop) {
        CFRunLoopStop(runLoop);
    }
}

static void deregisterSystemPowerNotifications() {
    if (runLoop && notifyPort) {
        CFRunLoopRemoveSource(runLoop, IONotificationPortGetRunLoopSource(notifyPort), kCFRunLoopDefaultMode);
    }
    if (notifierObject) {
        IODeregisterForSystemPower(&notifierObject);
    }
    if (notifyPort) {
        IONotificationPortDestroy(notifyPort);
    }
    if (rootPort) {
        IOServiceClose(rootPort);
    }
}
*/
import "C"

import (
	"fmt"
	"sync"
)

// SystemEventType represents a system event type
type SystemEventType int

const (
	SystemEventSleep SystemEventType = iota
	SystemEventWake
)

// SystemEventHandler handles system events
type SystemEventHandler interface {
	OnSystemSleep()
	OnSystemWake()
}

// systemEventManager manages system power events
type systemEventManager struct {
	mu       sync.Mutex
	handlers []SystemEventHandler
	running  bool
}

var (
	eventManager     *systemEventManager
	eventManagerOnce sync.Once
)

// getSystemEventManager returns the singleton system event manager
func getSystemEventManager() *systemEventManager {
	eventManagerOnce.Do(func() {
		eventManager = &systemEventManager{
			handlers: make([]SystemEventHandler, 0),
		}
	})
	return eventManager
}

// RegisterSystemEventHandler registers a handler for system events
func RegisterSystemEventHandler(handler SystemEventHandler) error {
	manager := getSystemEventManager()
	manager.mu.Lock()
	defer manager.mu.Unlock()

	manager.handlers = append(manager.handlers, handler)

	// Start monitoring if not already running
	if !manager.running {
		if err := manager.start(); err != nil {
			return fmt.Errorf("failed to start system event monitoring: %w", err)
		}
	}

	return nil
}

// UnregisterSystemEventHandler unregisters a handler
func UnregisterSystemEventHandler(handler SystemEventHandler) {
	manager := getSystemEventManager()
	manager.mu.Lock()
	defer manager.mu.Unlock()

	// Remove handler from list
	newHandlers := make([]SystemEventHandler, 0, len(manager.handlers))
	for _, h := range manager.handlers {
		if h != handler {
			newHandlers = append(newHandlers, h)
		}
	}
	manager.handlers = newHandlers

	// Stop monitoring if no handlers remain
	if len(manager.handlers) == 0 && manager.running {
		manager.stop()
	}
}

// start starts monitoring system events
func (m *systemEventManager) start() error {
	if m.running {
		return nil
	}

	// Register for power notifications
	result := C.registerSystemPowerNotifications()
	if result != 0 {
		return fmt.Errorf("failed to register for system power notifications")
	}

	m.running = true

	// Run event loop in a separate goroutine
	go func() {
		C.runSystemEventLoop()
	}()

	return nil
}

// stop stops monitoring system events
func (m *systemEventManager) stop() {
	if !m.running {
		return
	}

	C.stopSystemEventLoop()
	C.deregisterSystemPowerNotifications()
	m.running = false
}

// notifyHandlers notifies all registered handlers of an event
func (m *systemEventManager) notifyHandlers(eventType SystemEventType) {
	m.mu.Lock()
	handlers := make([]SystemEventHandler, len(m.handlers))
	copy(handlers, m.handlers)
	m.mu.Unlock()

	for _, handler := range handlers {
		switch eventType {
		case SystemEventSleep:
			handler.OnSystemSleep()
		case SystemEventWake:
			handler.OnSystemWake()
		}
	}
}

//export systemEventCallback
func systemEventCallback(eventType C.int) {
	manager := getSystemEventManager()
	manager.notifyHandlers(SystemEventType(eventType))
}
