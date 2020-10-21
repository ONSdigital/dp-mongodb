// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package mock

import (
	"github.com/ONSdigital/dp-mongodb/health"
	"sync"
)

// Ensure, that SessionerMock does implement health.Sessioner.
// If this is not the case, regenerate this file with moq.
var _ health.Sessioner = &SessionerMock{}

// SessionerMock is a mock implementation of health.Sessioner.
//
//     func TestSomethingThatUsesSessioner(t *testing.T) {
//
//         // make and configure a mocked health.Sessioner
//         mockedSessioner := &SessionerMock{
//             CloseFunc: func()  {
// 	               panic("mock out the Close method")
//             },
//             CopyFunc: func() health.Sessioner {
// 	               panic("mock out the Copy method")
//             },
//             DBFunc: func(name string) health.Databaser {
// 	               panic("mock out the DB method")
//             },
//             PingFunc: func() error {
// 	               panic("mock out the Ping method")
//             },
//         }
//
//         // use mockedSessioner in code that requires health.Sessioner
//         // and then make assertions.
//
//     }
type SessionerMock struct {
	// CloseFunc mocks the Close method.
	CloseFunc func()

	// CopyFunc mocks the Copy method.
	CopyFunc func() health.Sessioner

	// DBFunc mocks the DB method.
	DBFunc func(name string) health.Databaser

	// PingFunc mocks the Ping method.
	PingFunc func() error

	// calls tracks calls to the methods.
	calls struct {
		// Close holds details about calls to the Close method.
		Close []struct {
		}
		// Copy holds details about calls to the Copy method.
		Copy []struct {
		}
		// DB holds details about calls to the DB method.
		DB []struct {
			// Name is the name argument value.
			Name string
		}
		// Ping holds details about calls to the Ping method.
		Ping []struct {
		}
	}
	lockClose sync.RWMutex
	lockCopy  sync.RWMutex
	lockDB    sync.RWMutex
	lockPing  sync.RWMutex
}

// Close calls CloseFunc.
func (mock *SessionerMock) Close() {
	if mock.CloseFunc == nil {
		panic("SessionerMock.CloseFunc: method is nil but Sessioner.Close was just called")
	}
	callInfo := struct {
	}{}
	mock.lockClose.Lock()
	mock.calls.Close = append(mock.calls.Close, callInfo)
	mock.lockClose.Unlock()
	mock.CloseFunc()
}

// CloseCalls gets all the calls that were made to Close.
// Check the length with:
//     len(mockedSessioner.CloseCalls())
func (mock *SessionerMock) CloseCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockClose.RLock()
	calls = mock.calls.Close
	mock.lockClose.RUnlock()
	return calls
}

// Copy calls CopyFunc.
func (mock *SessionerMock) Copy() health.Sessioner {
	if mock.CopyFunc == nil {
		panic("SessionerMock.CopyFunc: method is nil but Sessioner.Copy was just called")
	}
	callInfo := struct {
	}{}
	mock.lockCopy.Lock()
	mock.calls.Copy = append(mock.calls.Copy, callInfo)
	mock.lockCopy.Unlock()
	return mock.CopyFunc()
}

// CopyCalls gets all the calls that were made to Copy.
// Check the length with:
//     len(mockedSessioner.CopyCalls())
func (mock *SessionerMock) CopyCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockCopy.RLock()
	calls = mock.calls.Copy
	mock.lockCopy.RUnlock()
	return calls
}

// DB calls DBFunc.
func (mock *SessionerMock) DB(name string) health.Databaser {
	if mock.DBFunc == nil {
		panic("SessionerMock.DBFunc: method is nil but Sessioner.DB was just called")
	}
	callInfo := struct {
		Name string
	}{
		Name: name,
	}
	mock.lockDB.Lock()
	mock.calls.DB = append(mock.calls.DB, callInfo)
	mock.lockDB.Unlock()
	return mock.DBFunc(name)
}

// DBCalls gets all the calls that were made to DB.
// Check the length with:
//     len(mockedSessioner.DBCalls())
func (mock *SessionerMock) DBCalls() []struct {
	Name string
} {
	var calls []struct {
		Name string
	}
	mock.lockDB.RLock()
	calls = mock.calls.DB
	mock.lockDB.RUnlock()
	return calls
}

// Ping calls PingFunc.
func (mock *SessionerMock) Ping() error {
	if mock.PingFunc == nil {
		panic("SessionerMock.PingFunc: method is nil but Sessioner.Ping was just called")
	}
	callInfo := struct {
	}{}
	mock.lockPing.Lock()
	mock.calls.Ping = append(mock.calls.Ping, callInfo)
	mock.lockPing.Unlock()
	return mock.PingFunc()
}

// PingCalls gets all the calls that were made to Ping.
// Check the length with:
//     len(mockedSessioner.PingCalls())
func (mock *SessionerMock) PingCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockPing.RLock()
	calls = mock.calls.Ping
	mock.lockPing.RUnlock()
	return calls
}

// Ensure, that DatabaserMock does implement health.Databaser.
// If this is not the case, regenerate this file with moq.
var _ health.Databaser = &DatabaserMock{}

// DatabaserMock is a mock implementation of health.Databaser.
//
//     func TestSomethingThatUsesDatabaser(t *testing.T) {
//
//         // make and configure a mocked health.Databaser
//         mockedDatabaser := &DatabaserMock{
//             CollectionNamesFunc: func() ([]string, error) {
// 	               panic("mock out the CollectionNames method")
//             },
//         }
//
//         // use mockedDatabaser in code that requires health.Databaser
//         // and then make assertions.
//
//     }
type DatabaserMock struct {
	// CollectionNamesFunc mocks the CollectionNames method.
	CollectionNamesFunc func() ([]string, error)

	// calls tracks calls to the methods.
	calls struct {
		// CollectionNames holds details about calls to the CollectionNames method.
		CollectionNames []struct {
		}
	}
	lockCollectionNames sync.RWMutex
}

// CollectionNames calls CollectionNamesFunc.
func (mock *DatabaserMock) CollectionNames() ([]string, error) {
	if mock.CollectionNamesFunc == nil {
		panic("DatabaserMock.CollectionNamesFunc: method is nil but Databaser.CollectionNames was just called")
	}
	callInfo := struct {
	}{}
	mock.lockCollectionNames.Lock()
	mock.calls.CollectionNames = append(mock.calls.CollectionNames, callInfo)
	mock.lockCollectionNames.Unlock()
	return mock.CollectionNamesFunc()
}

// CollectionNamesCalls gets all the calls that were made to CollectionNames.
// Check the length with:
//     len(mockedDatabaser.CollectionNamesCalls())
func (mock *DatabaserMock) CollectionNamesCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockCollectionNames.RLock()
	calls = mock.calls.CollectionNames
	mock.lockCollectionNames.RUnlock()
	return calls
}