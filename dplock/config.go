package dplock

import "time"

// Default config values
const (
	DefaultTTL               = 30
	DefaultPurgerPeriod      = 5 * time.Minute
	DefaultAcquirePeriod     = 250 * time.Millisecond
	DefaultUnlockPeriod      = 5 * time.Millisecond
	DefaultAcquireMaxRetries = 10
	DefaultUnlockMaxRetries  = 100
)

// Config is a lock configuration
type Config struct {
	TTL               uint          // TTL is the 'time to live' for a lock in number of seconds
	PurgerPeriod      time.Duration // PurgerPeriod is the time period between expired lock purges
	AcquirePeriod     time.Duration // AcquirePeriod is the time period between acquire lock retries
	UnlockPeriod      time.Duration // UnlockPeriod is the time period between Unlock lock retries
	AcquireMaxRetries int           // AcquireMaxRetries is the maximum number of locking retries by the Acquire lock, discounting the first attempt
	UnlockMaxRetries  int           // UnlockMaxRetries is the maximum number of unlocking retries by the Unlock lock, discounting the first attempt
}

// ConfigOverride is a config with pointer values, which are used to override values (it not nil)
type ConfigOverride struct {
	TTL               *uint
	PurgerPeriod      *time.Duration
	AcquirePeriod     *time.Duration
	UnlockPeriod      *time.Duration
	AcquireMaxRetries *int
	UnlockMaxRetries  *int
}

// GetConfig returns a full config, containing any value provided by configOverrides,
// and the default values otherwise
func GetConfig(cfgOverride *ConfigOverride) Config {
	// default values
	cfg := Config{
		TTL:               DefaultTTL,
		PurgerPeriod:      DefaultPurgerPeriod,
		AcquirePeriod:     DefaultAcquirePeriod,
		UnlockPeriod:      DefaultUnlockPeriod,
		AcquireMaxRetries: DefaultAcquireMaxRetries,
		UnlockMaxRetries:  DefaultUnlockMaxRetries,
	}
	// default any provided non-nil value:
	if cfgOverride != nil {
		if cfgOverride.TTL != nil {
			cfg.TTL = *cfgOverride.TTL
		}
		if cfgOverride.PurgerPeriod != nil {
			cfg.PurgerPeriod = *cfgOverride.PurgerPeriod
		}
		if cfgOverride.AcquirePeriod != nil {
			cfg.AcquirePeriod = *cfgOverride.AcquirePeriod
		}
		if cfgOverride.UnlockPeriod != nil {
			cfg.UnlockPeriod = *cfgOverride.UnlockPeriod
		}
		if cfgOverride.AcquireMaxRetries != nil {
			cfg.AcquireMaxRetries = *cfgOverride.AcquireMaxRetries
		}
		if cfgOverride.UnlockMaxRetries != nil {
			cfg.UnlockMaxRetries = *cfgOverride.UnlockMaxRetries
		}
	}

	return cfg
}
