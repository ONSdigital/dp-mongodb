package dplock

import (
	"errors"
	"fmt"
	"time"
)

// Default config values
const (
	DefaultTTL                           = 30
	DefaultPurgerPeriod                  = 5 * time.Minute
	DefaultAcquireMinPeriodMillis        = 50
	DefaultAcquireMaxPeriodMillis        = 150
	DefaultAcquireRetryTimeout           = 10 * time.Second
	DefaultUnlockMinPeriodMillis         = 5
	DefaultUnlockMaxPeriodMillis         = 10
	DefaultUnlockRetryTimeout            = 5 * time.Second
	DefaultTimeThresholdSinceLastRelease = 100 * time.Millisecond
	DefaultUsageSleep                    = 50 * time.Millisecond
	DefaultMaxCount                      = 10
)

const MinAllowedPurgerPeriod = time.Second

// Config is a lock configuration
type Config struct {
	TTL                           uint          // TTL is the 'time to live' for a lock in number of seconds, note that expred locks will be cleaned up by the purger (so the worst case scenario is that a lock is cleaned up after TTL + PurgerPeriod)
	PurgerPeriod                  time.Duration // PurgerPeriod is the time period between expired lock purges
	AcquireMinPeriodMillis        uint          // AcquireMinPeriod is the minimum time period between acquire lock retries [ms]
	AcquireMaxPeriodMillis        uint          // AcquireMinPeriod is the maximum time period between acquire lock retries [ms]
	AcquireRetryTimeout           time.Duration // AcquireRetryTimeout is the maximum time period that locking will be retried, after the first attempt has failed
	UnlockMinPeriodMillis         uint          // UnlockMinPeriod is the minimum time period between Unlock retries [ms]
	UnlockMaxPeriodMillis         uint          // UnlockMaxPeriod is the maximum time period between Unlock retries [ms]
	UnlockRetryTimeout            time.Duration // UnlockRetryTimeout is the maximum time period that unlocking will be retried, after the first attempt has failed
	TimeThresholdSinceLastRelease time.Duration // TimeThresholdSinceLastRelease is the maximum period of time since the last time a lock was released for which the 'usage' counter will be increased
	UsageSleep                    time.Duration // UsageSleep is the period of time a caller will sleep after a sequence of MaxCount acquire and releases for a particular lock within TimeThresholdSinceLastRelease between releases and acquires.
	MaxCount                      uint          // MaxCount is the number of consecutive times a lock is released and re-acquired within a TimeThresholdSinceLastRelease period, for which a sleep will be triggered in order to prevent the caller from monopolising the lock.
}

// ConfigOverride is a config with pointer values, which are used to override values (it not nil)
type ConfigOverride struct {
	TTL                           *uint
	PurgerPeriod                  *time.Duration
	AcquireMinPeriodMillis        *uint
	AcquireMaxPeriodMillis        *uint
	AcquireRetryTimeout           *time.Duration
	UnlockMinPeriodMillis         *uint
	UnlockMaxPeriodMillis         *uint
	UnlockRetryTimeout            *time.Duration
	TimeThresholdSinceLastRelease *time.Duration
	UsageSleep                    *time.Duration
	MaxCount                      *uint
}

// GetConfig returns a full config, containing any value provided by configOverrides,
// and the default values otherwise
func GetConfig(cfgOverride *ConfigOverride) Config {
	// default values
	cfg := Config{
		TTL:                           DefaultTTL,
		PurgerPeriod:                  DefaultPurgerPeriod,
		AcquireMinPeriodMillis:        DefaultAcquireMinPeriodMillis,
		AcquireMaxPeriodMillis:        DefaultAcquireMaxPeriodMillis,
		AcquireRetryTimeout:           DefaultAcquireRetryTimeout,
		UnlockMinPeriodMillis:         DefaultUnlockMinPeriodMillis,
		UnlockMaxPeriodMillis:         DefaultUnlockMaxPeriodMillis,
		UnlockRetryTimeout:            DefaultUnlockRetryTimeout,
		TimeThresholdSinceLastRelease: DefaultTimeThresholdSinceLastRelease,
		UsageSleep:                    DefaultUsageSleep,
		MaxCount:                      DefaultMaxCount,
	}
	// override any provided non-nil value:
	if cfgOverride != nil {
		if cfgOverride.TTL != nil {
			cfg.TTL = *cfgOverride.TTL
		}
		if cfgOverride.PurgerPeriod != nil {
			cfg.PurgerPeriod = *cfgOverride.PurgerPeriod
		}
		if cfgOverride.AcquireMinPeriodMillis != nil {
			cfg.AcquireMinPeriodMillis = *cfgOverride.AcquireMinPeriodMillis
		}
		if cfgOverride.AcquireMaxPeriodMillis != nil {
			cfg.AcquireMaxPeriodMillis = *cfgOverride.AcquireMaxPeriodMillis
		}
		if cfgOverride.AcquireRetryTimeout != nil {
			cfg.AcquireRetryTimeout = *cfgOverride.AcquireRetryTimeout
		}
		if cfgOverride.UnlockMinPeriodMillis != nil {
			cfg.UnlockMinPeriodMillis = *cfgOverride.UnlockMinPeriodMillis
		}
		if cfgOverride.UnlockMaxPeriodMillis != nil {
			cfg.UnlockMaxPeriodMillis = *cfgOverride.UnlockMaxPeriodMillis
		}
		if cfgOverride.UnlockRetryTimeout != nil {
			cfg.UnlockRetryTimeout = *cfgOverride.UnlockRetryTimeout
		}
		if cfgOverride.TimeThresholdSinceLastRelease != nil {
			cfg.TimeThresholdSinceLastRelease = *cfgOverride.TimeThresholdSinceLastRelease
		}
		if cfgOverride.UsageSleep != nil {
			cfg.UsageSleep = *cfgOverride.UsageSleep
		}
		if cfgOverride.MaxCount != nil {
			cfg.MaxCount = *cfgOverride.MaxCount
		}
	}

	return cfg
}

// Validate checks that the config values will not result in any unexpected behavior
func (c *Config) Validate() error {
	if c.PurgerPeriod < MinAllowedPurgerPeriod {
		return fmt.Errorf("the minimum allowed purger period is %s", MinAllowedPurgerPeriod)
	}
	if c.AcquireMaxPeriodMillis <= c.AcquireMinPeriodMillis {
		return errors.New("acquire max period must be greater than acquire min period")
	}
	if c.UnlockMaxPeriodMillis <= c.UnlockMinPeriodMillis {
		return errors.New("unlock max period must be greater than unlock min period")
	}
	return nil
}
