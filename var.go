package stm

import (
	"sync"
	"sync/atomic"
)

// Holds an STM variable.
type Var [T any]struct {
	value    atomic.Value
	watchers sync.Map
	mu       sync.Mutex
}

func (v *Var[T]) getValue() *atomic.Value{
	return &v.value
}

func (v *Var[T]) getWatchers() *sync.Map{
	return &v.watchers
}

func (v *Var[T]) getLock() *sync.Mutex {
	return &v.mu
}

func (v *Var[T]) changeValue(new interface{}) {
	old := v.value.Load().(VarValue)
	newVarValue := old.Set(new)
	v.value.Store(newVarValue)
	if old.Changed(newVarValue) {
		go v.wakeWatchers(newVarValue)
	}
}

func (v *Var[T]) wakeWatchers(new VarValue) {
	v.watchers.Range(func(k, _ interface{}) bool {
		tx := k.(*Tx)
		// We have to lock here to ensure that the Tx is waiting before we signal it. Otherwise we
		// could signal it before it goes to sleep and it will miss the notification.
		tx.mu.Lock()
		if read := tx.reads[v]; read != nil && read.Changed(new) {
			tx.cond.Broadcast()
			for !tx.waiting && !tx.completed {
				tx.cond.Wait()
			}
		}
		tx.mu.Unlock()
		return !v.value.Load().(VarValue).Changed(new)
	})
}

type varSnapshot struct {
	val     interface{}
	version uint64
}

// Returns a new STM variable.
func NewVar[T any](val T) *Var[T] {
	v := &Var[T]{}
	v.value.Store(versionedValue{
		value: val,
	})
	return v
}

func NewCustomVar[T any](val T, changed func(T, T) bool) *Var[T] {
	v := &Var[T]{}
	v.value.Store(customVarValue{
		value:   val,
		changed: changed,
	})
	return v
}

func NewBuiltinEqVar[T any](val T) *Var[T] {
	return NewCustomVar(val, func(a, b T) bool {
		return a != b
	})
}
