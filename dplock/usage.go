package dplock

import (
	"time"
)

// Usages is a 2D map to keep track of lock usages by resourceID and owner
type Usages map[string]map[string]*Usage

// Usage keeps track of locks that have been acquired by a particular caller (lock owner) and resource
type Usage struct {
	Count    uint      // counter for the number of times that a lock has been acquired on the first attempt within a short period of time after being released
	Released time.Time // timestamp for the last time that a lock was released
}

// getUsage returns the Usage for the provided resourceID and owner, if it exists
func (u Usages) getUsage(resourceID, owner string) (*Usage, bool) {
	resUsages, found := u[resourceID]
	if !found {
		return nil, false
	}
	usage, found := resUsages[owner]
	return usage, found
}

// SetCount increases the counter if the lock has been previously released in the last 'timeThresholdSinceLastRelease'
// otherwise it resets the counter to 0
// if the Usage did not exist in the map, it will be created
func (u Usages) SetCount(cfg *Config, resourceID, owner string) {
	resUsages, found := u[resourceID]
	if !found {
		resUsages = map[string]*Usage{}
		u[resourceID] = resUsages
	}
	usage, found := resUsages[owner]
	if !found {
		usage = &Usage{}
		resUsages[owner] = usage
	}
	if usage.Released.IsZero() || time.Since(usage.Released) <= cfg.TimeThresholdSinceLastRelease {
		usage.Count++ // increase count because the lock was released by the same caller a short period of time ago
	} else {
		usage.Count = 0 // reset count because the lock was released by the same caller a long period of time ago
	}
}

// WaitIfNeeded sleeps for 'usageSleep' time if the provided resource has been locked by the provided owner at least MaxCount times,
// with a period of time smaller than 'timeThresholdSinceLastRelease' between releasing and re-acquiring the lock for all times.
// After sleeping, the counter is reset to 0
func (u Usages) WaitIfNeeded(cfg *Config, resourceID, owner string) {
	usage, found := u.getUsage(resourceID, owner)
	if !found {
		return
	}
	if usage.Count >= cfg.MaxCount && time.Since(usage.Released) <= cfg.TimeThresholdSinceLastRelease {
		Sleep(cfg.UsageSleep)
		usage.Count = 0
	}
}

var Sleep = func(d time.Duration) {
	time.Sleep(d)
}

// SetReleased sets the provided released timestamp for the Usage of the provided resource and lock.
// Only if the usage already exists in the map
func (u Usages) SetReleased(resourceID, owner string, releasedTime time.Time) {
	usage, found := u.getUsage(resourceID, owner)
	if !found {
		return
	}
	usage.Released = releasedTime
}

// Remove deletes the Usage for the provided ResourceID and owner, if it exists.
// If the owner was the last one for a resourceID, that map will also be deleted.
func (u Usages) Remove(resourceID, owner string) {
	if _, found := u.getUsage(resourceID, owner); !found {
		return
	}
	delete(u[resourceID], owner) // remove item from innter map
	if len(u[resourceID]) == 0 {
		delete(u, resourceID) // remove outter map if it was the last item in it
	}
}

// Purge removes all the Usages that
func (u Usages) Purge(cfg *Config) {
	// check what items can be removed
	toRemove := [][]string{}
	for resourceID, resUsages := range u {
		for owner, usage := range resUsages {
			if time.Since(usage.Released) > cfg.TimeThresholdSinceLastRelease {
				toRemove = append(toRemove, []string{resourceID, owner})
			}
		}
	}

	// remove the items
	for _, pair := range toRemove {
		u.Remove(pair[0], pair[1])
	}
}
