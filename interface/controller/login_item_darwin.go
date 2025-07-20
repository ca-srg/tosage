//go:build darwin
// +build darwin

package controller

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation

#import <Foundation/Foundation.h>

// Simple file-based login item management for macOS
// This creates/removes a LaunchAgent plist file

static int addLoginItem(const char *appPath) {
    @autoreleasepool {
        NSString *path = [NSString stringWithUTF8String:appPath];
        NSString *appName = [[path lastPathComponent] stringByDeletingPathExtension];

        // Get the user's LaunchAgents directory
        NSArray *paths = NSSearchPathForDirectoriesInDomains(NSLibraryDirectory, NSUserDomainMask, YES);
        NSString *libraryPath = [paths objectAtIndex:0];
        NSString *launchAgentsPath = [libraryPath stringByAppendingPathComponent:@"LaunchAgents"];

        // Create LaunchAgents directory if it doesn't exist
        NSFileManager *fileManager = [NSFileManager defaultManager];
        [fileManager createDirectoryAtPath:launchAgentsPath withIntermediateDirectories:YES attributes:nil error:nil];

        // Create plist file path
        NSString *plistName = [NSString stringWithFormat:@"com.tosage.%@.plist", appName];
        NSString *plistPath = [launchAgentsPath stringByAppendingPathComponent:plistName];

        // Create plist content
        NSDictionary *plistDict = @{
            @"Label": [NSString stringWithFormat:@"com.tosage.%@", appName],
            @"ProgramArguments": @[path, @"--daemon"],
            @"RunAtLoad": @YES,
            @"KeepAlive": @NO
        };

        // Write plist file
        BOOL success = [plistDict writeToFile:plistPath atomically:YES];
        return success ? 0 : -1;
    }
}

static int removeLoginItem(const char *appPath) {
    @autoreleasepool {
        NSString *path = [NSString stringWithUTF8String:appPath];
        NSString *appName = [[path lastPathComponent] stringByDeletingPathExtension];

        // Get the user's LaunchAgents directory
        NSArray *paths = NSSearchPathForDirectoriesInDomains(NSLibraryDirectory, NSUserDomainMask, YES);
        NSString *libraryPath = [paths objectAtIndex:0];
        NSString *launchAgentsPath = [libraryPath stringByAppendingPathComponent:@"LaunchAgents"];

        // Create plist file path
        NSString *plistName = [NSString stringWithFormat:@"com.tosage.%@.plist", appName];
        NSString *plistPath = [launchAgentsPath stringByAppendingPathComponent:plistName];

        // Remove plist file
        NSFileManager *fileManager = [NSFileManager defaultManager];
        NSError *error = nil;
        BOOL success = [fileManager removeItemAtPath:plistPath error:&error];

        return success ? 0 : -1;
    }
}

static int isLoginItem(const char *appPath) {
    @autoreleasepool {
        NSString *path = [NSString stringWithUTF8String:appPath];
        NSString *appName = [[path lastPathComponent] stringByDeletingPathExtension];

        // Get the user's LaunchAgents directory
        NSArray *paths = NSSearchPathForDirectoriesInDomains(NSLibraryDirectory, NSUserDomainMask, YES);
        NSString *libraryPath = [paths objectAtIndex:0];
        NSString *launchAgentsPath = [libraryPath stringByAppendingPathComponent:@"LaunchAgents"];

        // Create plist file path
        NSString *plistName = [NSString stringWithFormat:@"com.tosage.%@.plist", appName];
        NSString *plistPath = [launchAgentsPath stringByAppendingPathComponent:plistName];

        // Check if plist file exists
        NSFileManager *fileManager = [NSFileManager defaultManager];
        return [fileManager fileExistsAtPath:plistPath] ? 1 : 0;
    }
}
*/
import "C"

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"
)

// LoginItemManager manages login items on macOS
type LoginItemManager struct {
	appPath string
}

// NewLoginItemManager creates a new login item manager
func NewLoginItemManager() (*LoginItemManager, error) {
	// Get the path to the current executable
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve any symlinks
	realPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	return &LoginItemManager{
		appPath: realPath,
	}, nil
}

// AddToLoginItems adds the application to login items
func (l *LoginItemManager) AddToLoginItems() error {
	cPath := C.CString(l.appPath)
	defer C.free(unsafe.Pointer(cPath))

	result := C.addLoginItem(cPath)
	if result != 0 {
		return fmt.Errorf("failed to add login item")
	}

	return nil
}

// RemoveFromLoginItems removes the application from login items
func (l *LoginItemManager) RemoveFromLoginItems() error {
	cPath := C.CString(l.appPath)
	defer C.free(unsafe.Pointer(cPath))

	result := C.removeLoginItem(cPath)
	if result != 0 {
		return fmt.Errorf("failed to remove login item")
	}

	return nil
}

// IsLoginItem checks if the application is in login items
func (l *LoginItemManager) IsLoginItem() (bool, error) {
	cPath := C.CString(l.appPath)
	defer C.free(unsafe.Pointer(cPath))

	result := C.isLoginItem(cPath)
	return result == 1, nil
}

// SetLoginItem sets the login item status
func (l *LoginItemManager) SetLoginItem(enabled bool) error {
	if enabled {
		return l.AddToLoginItems()
	}
	return l.RemoveFromLoginItems()
}
