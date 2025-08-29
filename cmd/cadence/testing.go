// helper code for unit testing 

package main

import (
    "sync"
)

// helper function to count sync map items 
func CountSyncMapItems(m *sync.Map) int {
    count := 0
    m.Range(func(key, value interface{}) bool {
        count++
        return true
    })
    return count
}