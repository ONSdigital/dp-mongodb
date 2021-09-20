package dplock

import (
	"sync"
	"time"
)

// Usages is a 2D map to keep track of lock usages by resourceID and unique caller names
type Usages struct {
	UsagesMap map[string]map[string]*Usage
	mutex     *sync.RWMutex
	cfg       *Config
}

// Usage keeps track of locks that have been acquired by a particular caller and resource
type Usage struct {
	Count    uint      // counter for the number of times that a lock has been acquired on the first attempt within a short period of time after being released
	Released time.Time // timestamp for the last time that a lock was released
}

// NewUsages returns a new Usages struct with the provided config and a new map and mutex
func NewUsages(cfg *Config) Usages {
	return Usages{
		UsagesMap: map[string]map[string]*Usage{},
		mutex:     &sync.RWMutex{},
		cfg:       cfg,
	}
}

// getUsage returns the Usage for the provided resourceID and uniqueCallerName, only if it exists.
// note that this is an internal method that does not acquire the mutex
func (u Usages) getUsage(resourceID, uniqueCallerName string) (*Usage, bool) {
	resUsages, found := u.UsagesMap[resourceID]
	if !found {
		return nil, false
	}
	usage, found := resUsages[uniqueCallerName]
	return usage, found
}

// getOrCreateUsage returns the Usage for the provided resourceID and unique caller name,
// if the usage and/or internal usage map did not exist, they will be created.
// note that this is an internal method that does not acquire the mutex
func (u Usages) getOrCreateUsage(resourceID, uniqueCallerName string) *Usage {
	// get map of usages by uniqueCallerName for the provided resource - create it if it does not exist
	resUsages, found := u.UsagesMap[resourceID]
	if !found {
		resUsages = map[string]*Usage{}
		u.UsagesMap[resourceID] = resUsages
	}

	// get Usage for the provided uniqueCallerName - create it if it does not exist
	usage, found := resUsages[uniqueCallerName]
	if !found {
		usage = &Usage{}
		resUsages[uniqueCallerName] = usage
	}
	return usage
}

// SetCount increases the counter if the lock has been previously released in the last 'timeThresholdSinceLastRelease'
// otherwise it resets the counter to 0
// if the Usage did not exist in the map, it will be created.
// This method is executed inside the Usages mutex
func (u Usages) SetCount(resourceID, uniqueCallerName string) {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	// get usage, or create a new one if it did not exist
	usage := u.getOrCreateUsage(resourceID, uniqueCallerName)

	// Increase the count if it has never been released, or it was released recently (earlier than TimeThresholdSinceLastRelease ago)
	if usage.Released.IsZero() || time.Since(usage.Released) <= u.cfg.TimeThresholdSinceLastRelease {
		usage.Count++
	} else {
		usage.Count = 0 // reset count because the lock was released by the same caller a long period of time ago
	}
}

// WaitIfNeeded sleeps for 'usageSleep' time if the provided resource has been locked by the provided unique caller name at least MaxCount times,
// with a period of time smaller than 'timeThresholdSinceLastRelease' between releasing and re-acquiring the lock for all times.
// After sleeping, the counter is reset to 0 (if the usage was purged, it will be re-created)
// This method is executed inside the Usages mutex, except the Sleep
func (u Usages) WaitIfNeeded(resourceID, uniqueCallerName string) {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	// get usage, if it is not found, return (we are sure we do not need to sleep)
	usage, found := u.getUsage(resourceID, uniqueCallerName)
	if !found {
		return
	}

	// check if the last released time is recent and maxCount has been achieved
	if usage.Count >= u.cfg.MaxCount && time.Since(usage.Released) <= u.cfg.TimeThresholdSinceLastRelease {

		// Sleep without keeping the mutex
		u.mutex.Unlock()
		Sleep(u.cfg.UsageSleep)
		u.mutex.Lock()

		// check if the usage is still in the map (it might have been purged while we were sleeping)
		usage = u.getOrCreateUsage(resourceID, uniqueCallerName)
		usage.Count = 0 // Reset the counter
	}
}

var Sleep = func(d time.Duration) {
	time.Sleep(d)
}

// SetReleased sets the provided released timestamp for the Usage of the provided resource and lock.
// Only if the usage already exists in the map.
// This method is executed inside the Usages mutex
func (u Usages) SetReleased(resourceID, uniqueCallerName string, releasedTime time.Time) {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	// get usage and set the releasedTime only if we find it
	usage, found := u.getUsage(resourceID, uniqueCallerName)
	if !found {
		return
	}
	usage.Released = releasedTime
}

// Remove deletes the Usage for the provided ResourceID and uniqueCallerName, if it exists.
// If the uniqueCallerName was the last one for a resourceID, that map will also be deleted.
// This method is executed inside the Usages mutex
func (u Usages) Remove(resourceID, uniqueCallerName string) {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	// call remove once we have the mutex
	u.remove(resourceID, uniqueCallerName)
}

// remove deletes the Usage for the provided ResourceID and uniqueCallerName, if it exists.
// If the uniqueCallerName was the last one for a resourceID, that map will also be deleted.
// note that this is an internal method that does not acquire the mutex
func (u Usages) remove(resourceID, uniqueCallerName string) {
	if _, found := u.getUsage(resourceID, uniqueCallerName); !found {
		return
	}
	delete(u.UsagesMap[resourceID], uniqueCallerName) // remove item from innter map
	if len(u.UsagesMap[resourceID]) == 0 {
		delete(u.UsagesMap, resourceID) // remove outter map if it was the last item in it
	}
}

// Purge removes all the Usages that have expired (last release happened earlier than TimeThresholdSinceLastRelease ago)
// This method is executed inside the Usages mutex
func (u Usages) Purge() {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	// check what items can be removed
	toRemove := [][]string{}
	for resourceID, resUsages := range u.UsagesMap {
		for uniqueCallerName, usage := range resUsages {
			if time.Since(usage.Released) > u.cfg.TimeThresholdSinceLastRelease {
				toRemove = append(toRemove, []string{resourceID, uniqueCallerName})
			}
		}
	}

	// remove the items
	for _, pair := range toRemove {
		u.remove(pair[0], pair[1])
	}
}
