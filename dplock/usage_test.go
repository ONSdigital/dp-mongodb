package dplock_test

import (
	"testing"
	"time"

	"github.com/ONSdigital/dp-mongodb/dplock"
	. "github.com/smartystreets/goconvey/convey"
)

var cfg = &dplock.Config{
	MaxCount:                      dplock.DefaultMaxCount,
	TimeThresholdSinceLastRelease: dplock.DefaultTimeThresholdSinceLastRelease,
	UsageSleep:                    dplock.DefaultUsageSleep,
}

func TestSetCount(t *testing.T) {

	Convey("Given a new Usages var", t, func() {
		u := dplock.NewUsages(cfg)

		Convey("Then SetCount creates the expected Usage and sets a value of 1 to the count", func() {
			u.SetCount(testResourceName, testOwner)
			So(u.UsagesMap, ShouldResemble, map[string]map[string]*dplock.Usage{
				testResourceName: {
					testOwner: {
						Count: 1,
					},
				},
			})
		})
	})

	Convey("Given a Usages var with a Released time more recent than 'timeThresholdSinceLastRelease' ago", t, func() {
		t0 := getUnexpiredTime()
		u := dplock.NewUsages(cfg)
		u.UsagesMap = map[string]map[string]*dplock.Usage{
			testResourceName: {
				testOwner: {
					Count:    3,
					Released: t0,
				},
			},
		}

		Convey("Then SetCount increases the count value by 1", func() {
			u.SetCount(testResourceName, testOwner)
			So(u.UsagesMap, ShouldResemble, map[string]map[string]*dplock.Usage{
				testResourceName: {
					testOwner: {
						Count:    4,
						Released: t0,
					},
				},
			})
		})
	})

	Convey("Given a Usages var with a Released time older than 'timeThresholdSinceLastRelease'", t, func() {
		t0 := getExpiredTime()
		u := dplock.NewUsages(cfg)
		u.UsagesMap = map[string]map[string]*dplock.Usage{
			testResourceName: {
				testOwner: {
					Count:    3,
					Released: t0,
				},
			},
		}

		Convey("Then SetCount is set to 0", func() {
			u.SetCount(testResourceName, testOwner)
			So(u.UsagesMap, ShouldResemble, map[string]map[string]*dplock.Usage{
				testResourceName: {
					testOwner: {
						Count:    0,
						Released: t0,
					},
				},
			})
		})
	})
}

func TestWaitIfNeeded(t *testing.T) {

	Convey("Given a mocked Sleep function", t, func() {
		slept := []time.Duration{}
		dplock.Sleep = func(d time.Duration) {
			slept = append(slept, d)
		}

		Convey("And an empty Usages", func() {
			u := dplock.Usages{}

			Convey("Then WaitIfNeeded does not sleep and does not modify the struct", func() {
				u.WaitIfNeeded(testResourceName, testOwner)
				So(slept, ShouldBeEmpty)
				So(u, ShouldResemble, dplock.Usages{})
			})

		})

		Convey("And a Usages that contains a non-expired Released time and a MaxCount count value", func() {
			t0 := getUnexpiredTime()
			u := dplock.NewUsages(cfg)
			u.UsagesMap = map[string]map[string]*dplock.Usage{
				testResourceName: {
					testOwner: {
						Count:    cfg.MaxCount,
						Released: t0,
					},
				},
			}

			Convey("Then WaitIfNeeded will sleep for the expected duration and resets the counter", func() {
				u.WaitIfNeeded(testResourceName, testOwner)
				So(slept, ShouldHaveLength, 1)
				So(slept[0], ShouldEqual, cfg.UsageSleep)
				So(u.UsagesMap, ShouldResemble, map[string]map[string]*dplock.Usage{
					testResourceName: {
						testOwner: {
							Count:    0,
							Released: t0,
						},
					},
				})
			})
		})

		Convey("And a Usages that contains an expired Released time and a MaxCount count value", func() {
			t0 := getExpiredTime()
			u := dplock.NewUsages(cfg)
			u.UsagesMap = map[string]map[string]*dplock.Usage{
				testResourceName: {
					testOwner: {
						Count:    cfg.MaxCount,
						Released: t0,
					},
				},
			}

			Convey("Then WaitIfNeeded does not sleep and does not reset the counter", func() {
				u.WaitIfNeeded(testResourceName, testOwner)
				So(slept, ShouldBeEmpty)
				So(u.UsagesMap, ShouldResemble, map[string]map[string]*dplock.Usage{
					testResourceName: {
						testOwner: {
							Count:    cfg.MaxCount,
							Released: t0,
						},
					},
				})
			})
		})

		Convey("And a Usages that contains a non-expired Released time and a count value lower than MaxCount", func() {
			t0 := getUnexpiredTime()
			u := dplock.NewUsages(cfg)
			u.UsagesMap = map[string]map[string]*dplock.Usage{
				testResourceName: {
					testOwner: {
						Count:    3,
						Released: t0,
					},
				},
			}

			Convey("Then WaitIfNeeded does not sleep and does not reset the counter", func() {
				u.WaitIfNeeded(testResourceName, testOwner)
				So(slept, ShouldBeEmpty)
				So(u.UsagesMap, ShouldResemble, map[string]map[string]*dplock.Usage{
					testResourceName: {
						testOwner: {
							Count:    3,
							Released: t0,
						},
					},
				})
			})
		})
	})
}

func TestSetReleased(t *testing.T) {

	Convey("Given an empty Usages", t, func() {
		u := dplock.Usages{}

		Convey("Then SetReleased does nothing", func() {
			u.SetReleased(testResourceName, testOwner, time.Now())
			So(u, ShouldResemble, dplock.Usages{})
		})
	})

	Convey("Given a Usages that contains an empty Usage for the resource and owner", t, func() {
		u := dplock.NewUsages(cfg)
		u.UsagesMap = map[string]map[string]*dplock.Usage{
			testResourceName: {
				testOwner: {},
			},
		}

		Convey("Then SetReleased overrides the released value", func() {
			t0 := time.Now()
			u.SetReleased(testResourceName, testOwner, t0)
			So(u.UsagesMap, ShouldResemble, map[string]map[string]*dplock.Usage{
				testResourceName: {
					testOwner: {
						Released: t0,
					},
				},
			})
		})
	})
}

func TestRemove(t *testing.T) {

	Convey("Given a Usages var", t, func() {
		t0 := time.Now()
		u := dplock.NewUsages(cfg)
		u.UsagesMap = map[string]map[string]*dplock.Usage{
			testResourceName: {
				testOwner: {
					Count:    3,
					Released: t0,
				},
				"otherOwner": {},
			},
		}

		Convey("Then removing an existing resource and owner results in the item being removed from the Usages inner map", func() {
			u.Remove(testResourceName, testOwner)
			So(u.UsagesMap, ShouldResemble, map[string]map[string]*dplock.Usage{
				testResourceName: {
					"otherOwner": {},
				},
			})

			Convey("Then removing the last resource and owner results in the whole inner map for the resource being removed", func() {
				u.Remove(testResourceName, "otherOwner")
				So(u.UsagesMap, ShouldResemble, map[string]map[string]*dplock.Usage{})
			})
		})

		Convey("Then removing an inexistent owner for an existing resource has no effect", func() {
			u.Remove(testResourceName, "wrong")
			So(u.UsagesMap, ShouldResemble, map[string]map[string]*dplock.Usage{
				testResourceName: {
					testOwner: {
						Count:    3,
						Released: t0,
					},
					"otherOwner": {},
				},
			})
		})

		Convey("Then removing an owner for an inexistent resource has no effect", func() {
			u.Remove("wrong", testOwner)
			So(u.UsagesMap, ShouldResemble, map[string]map[string]*dplock.Usage{
				testResourceName: {
					testOwner: {
						Count:    3,
						Released: t0,
					},
					"otherOwner": {},
				},
			})
		})
	})
}

func TestPurge(t *testing.T) {

	Convey("Given an empty Usages var", t, func() {
		u := dplock.Usages{}

		Convey("Then Purge has no effect", func() {
			u.Purge()
			So(u, ShouldResemble, dplock.Usages{})
		})
	})

	Convey("Given a Usages var with expired and unexpired Usages", t, func() {
		t0 := getUnexpiredTime()
		t1 := getExpiredTime()
		u := dplock.NewUsages(cfg)
		u.UsagesMap = map[string]map[string]*dplock.Usage{
			testResourceName: {
				testOwner:    {Released: t0},
				"otherOwner": {Released: t1}, // expired
			},
			"otherResource": {
				testOwner: {Released: t1}, //expired
			},
		}

		Convey("Then Purge removes the expired ones only", func() {
			u.Purge()
			So(u.UsagesMap, ShouldResemble, map[string]map[string]*dplock.Usage{
				testResourceName: {
					testOwner: {Released: t0},
				},
			})
		})
	})
}

// returns an expired time for testing
func getExpiredTime() time.Time {
	return time.Now().Add(-cfg.TimeThresholdSinceLastRelease)
}

// returns a non-expired time for testing
func getUnexpiredTime() time.Time {
	return time.Now().Add(time.Minute)
}
