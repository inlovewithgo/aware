package logging

import (
    "sync"
)

var (
    isEnabled = true
    mu        sync.RWMutex
)

func Enable() {
    mu.Lock()
    defer mu.Unlock()
    isEnabled = true
}

func Disable() {
    mu.Lock()
    defer mu.Unlock()
    isEnabled = false
}

func IsEnabled() bool {
    mu.RLock()
    defer mu.RUnlock()
    return isEnabled
}
