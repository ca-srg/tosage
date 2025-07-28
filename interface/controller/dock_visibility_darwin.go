//go:build darwin
// +build darwin

package controller

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework AppKit

#import <Foundation/Foundation.h>
#import <AppKit/AppKit.h>

void hideFromDock() {
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
}

void showInDock() {
    [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
}

int isDockVisible() {
    NSApplicationActivationPolicy policy = [NSApp activationPolicy];
    return policy == NSApplicationActivationPolicyRegular ? 1 : 0;
}

void initializeApplication() {
    [NSApplication sharedApplication];
}
*/
import "C"

// HideFromDock hides the application from the macOS Dock
func HideFromDock() {
	C.initializeApplication()
	C.hideFromDock()
}

// ShowInDock shows the application in the macOS Dock
func ShowInDock() {
	C.initializeApplication()
	C.showInDock()
}

// IsDockVisible returns whether the application is visible in the Dock
func IsDockVisible() bool {
	C.initializeApplication()
	return C.isDockVisible() == 1
}
