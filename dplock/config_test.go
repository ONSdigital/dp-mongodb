package dplock_test

import (
	"errors"
	"testing"
	"time"

	"github.com/ONSdigital/dp-mongodb/v2/dplock"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetConfig(t *testing.T) {

	Convey("Calling GetConfig without overrides results in the default config being returned", t, func() {
		cfg := dplock.GetConfig(nil)
		So(cfg, ShouldResemble, dplock.Config{
			TTL:                           dplock.DefaultTTL,
			PurgerPeriod:                  dplock.DefaultPurgerPeriod,
			AcquireMinPeriodMillis:        dplock.DefaultAcquireMinPeriodMillis,
			AcquireMaxPeriodMillis:        dplock.DefaultAcquireMaxPeriodMillis,
			AcquireRetryTimeout:           dplock.DefaultAcquireRetryTimeout,
			UnlockMinPeriodMillis:         dplock.DefaultUnlockMinPeriodMillis,
			UnlockMaxPeriodMillis:         dplock.DefaultUnlockMaxPeriodMillis,
			UnlockRetryTimeout:            dplock.DefaultUnlockRetryTimeout,
			TimeThresholdSinceLastRelease: dplock.DefaultTimeThresholdSinceLastRelease,
			UsageSleep:                    dplock.DefaultUsageSleep,
			MaxCount:                      dplock.DefaultMaxCount,
		})
	})

	Convey("Calling GetConfig with some overrides results in the default config with overrides being returned", t, func() {
		var (
			ttl            uint          = 123
			acquireTimeout time.Duration = 30 * time.Second
		)
		cfg := dplock.GetConfig(&dplock.ConfigOverride{
			TTL:                 &ttl,
			AcquireRetryTimeout: &acquireTimeout,
		})
		So(cfg, ShouldResemble, dplock.Config{
			TTL:                           ttl,
			PurgerPeriod:                  dplock.DefaultPurgerPeriod,
			AcquireMinPeriodMillis:        dplock.DefaultAcquireMinPeriodMillis,
			AcquireMaxPeriodMillis:        dplock.DefaultAcquireMaxPeriodMillis,
			AcquireRetryTimeout:           acquireTimeout,
			UnlockMinPeriodMillis:         dplock.DefaultUnlockMinPeriodMillis,
			UnlockMaxPeriodMillis:         dplock.DefaultUnlockMaxPeriodMillis,
			UnlockRetryTimeout:            dplock.DefaultUnlockRetryTimeout,
			TimeThresholdSinceLastRelease: dplock.DefaultTimeThresholdSinceLastRelease,
			UsageSleep:                    dplock.DefaultUsageSleep,
			MaxCount:                      dplock.DefaultMaxCount,
		})
	})

	Convey("Calling GetConfig with all overrides results in the fully overwritten being returned", t, func() {
		var (
			ttl                           uint          = 123
			purgerPeriod                  time.Duration = 10 * time.Second
			acquireMinMillis              uint          = 100
			acquireMaxMillis              uint          = 200
			acquireTimeout                time.Duration = 30 * time.Second
			unlockMinMillis               uint          = 10
			unlockMaxMillis               uint          = 20
			unlockTimeout                 time.Duration = 8 * time.Second
			timeThresholdSinceLastRelease time.Duration = 150 * time.Millisecond
			usageSleep                    time.Duration = 75 * time.Millisecond
			maxCount                      uint          = 15
		)
		cfg := dplock.GetConfig(&dplock.ConfigOverride{
			TTL:                           &ttl,
			PurgerPeriod:                  &purgerPeriod,
			AcquireMinPeriodMillis:        &acquireMinMillis,
			AcquireMaxPeriodMillis:        &acquireMaxMillis,
			AcquireRetryTimeout:           &acquireTimeout,
			UnlockMinPeriodMillis:         &unlockMinMillis,
			UnlockMaxPeriodMillis:         &unlockMaxMillis,
			UnlockRetryTimeout:            &unlockTimeout,
			TimeThresholdSinceLastRelease: &timeThresholdSinceLastRelease,
			UsageSleep:                    &usageSleep,
			MaxCount:                      &maxCount,
		})

		So(cfg, ShouldResemble, dplock.Config{
			TTL:                           ttl,
			PurgerPeriod:                  purgerPeriod,
			AcquireMinPeriodMillis:        acquireMinMillis,
			AcquireMaxPeriodMillis:        acquireMaxMillis,
			AcquireRetryTimeout:           acquireTimeout,
			UnlockMinPeriodMillis:         unlockMinMillis,
			UnlockMaxPeriodMillis:         unlockMaxMillis,
			UnlockRetryTimeout:            unlockTimeout,
			TimeThresholdSinceLastRelease: timeThresholdSinceLastRelease,
			UsageSleep:                    usageSleep,
			MaxCount:                      maxCount,
		})
	})
}

func TestValidate(t *testing.T) {

	Convey("A Config with a purger period lower than the minimum allowed fails to validate", t, func() {
		cfg := dplock.Config{
			PurgerPeriod: time.Millisecond,
		}
		err := cfg.Validate()
		So(err, ShouldResemble, errors.New("the minimum allowed purger period is 1s"))
	})

	Convey("A Config with AcquireMinPeriodMillis > AcquireMaxPeriodMillis fails to validate", t, func() {
		cfg := dplock.Config{
			PurgerPeriod:           time.Second,
			AcquireMinPeriodMillis: 100,
			AcquireMaxPeriodMillis: 99,
		}
		err := cfg.Validate()
		So(err, ShouldResemble, errors.New("acquire max period must be greater than acquire min period"))
	})

	Convey("A Config with UnlockMinPeriodMillis > UnlockMaxPeriodMillis fails to validate", t, func() {
		cfg := dplock.Config{
			PurgerPeriod:           time.Second,
			AcquireMinPeriodMillis: 100,
			AcquireMaxPeriodMillis: 101,
			UnlockMinPeriodMillis:  100,
			UnlockMaxPeriodMillis:  99,
		}
		err := cfg.Validate()
		So(err, ShouldResemble, errors.New("unlock max period must be greater than unlock min period"))
	})

	Convey("A valid Config succeeds validation", t, func() {
		cfg := dplock.Config{
			PurgerPeriod:           time.Second,
			AcquireMinPeriodMillis: 100,
			AcquireMaxPeriodMillis: 101,
			UnlockMinPeriodMillis:  100,
			UnlockMaxPeriodMillis:  101,
		}
		err := cfg.Validate()
		So(err, ShouldBeNil)
	})

	Convey("The default config succeeds validation", t, func() {
		cfg := dplock.GetConfig(nil)
		err := cfg.Validate()
		So(err, ShouldBeNil)
	})
}
