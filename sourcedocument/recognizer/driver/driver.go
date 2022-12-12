package driver

import (
	"fmt"
	"sort"
	"sync"
)

import (
	"github.com/chamzzzzzz/accounting/sourcedocument/types"
)

type Recognizer interface {
	Recognize(sourcefrom any) (*types.Source, error)
	Driver() Driver
}

type Driver interface {
	Open(paramName string) (Recognizer, error)
}

var (
	drivers = make(map[string]Driver)
	mu      sync.RWMutex
)

func Register(name string, driver Driver) {
	mu.Lock()
	defer mu.Unlock()
	if driver == nil {
		panic("accounting/sourcedocument/recognizer/driver: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("accounting/sourcedocument/recognizer/driver: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

func Drivers() []string {
	mu.RLock()
	defer mu.RUnlock()
	list := make([]string, 0, len(drivers))
	for name := range drivers {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

func Open(name, paramName string) (Recognizer, error) {
	mu.RLock()
	driver, ok := drivers[name]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown driver %q (forgotten import?)", name)
	}
	return driver.Open(paramName)
}
