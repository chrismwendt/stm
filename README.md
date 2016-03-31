# stm

[![GoDoc](https://godoc.org/github.com/lukechampine/stm?status.svg)](https://godoc.org/github.com/lukechampine/stm)

Package `stm` provides [Software Transactional Memory](https://en.wikipedia.org/wiki/Software_transactional_memory) operations for Go. This is
an alternative to the standard way of writing concurrent code (channels and
mutexes). STM makes it easy to perform arbitrarily complex operations in an
atomic fashion. One of its primary advantages over traditional locking is that
STM transactions are composable, whereas locking functions are not -- the
composition will either deadlock or release the lock between functions (making
it non-atomic).

The `stm` API tries to mimic that of Haskell's [`Control.Concurrent.STM`](https://hackage.haskell.org/package/stm-2.4.4.1/docs/Control-Concurrent-STM.html), but
this is not entirely possible due to Go's type system; we are forced to use
`interface{}` and type assertions. Furthermore, Haskell can enforce at compile
time that STM variables are not modified outside the STM monad. This is not
possible in Go, so be especially careful when using pointers in your STM code.
Another significant departure is that `stm.Atomically` does not return a value.
This shortens transaction code a bit, but I'm not 100% it's the right decision.
(The alternative would be for every transaction function to return an `interface{}`.)

It remains to be seen whether this style of concurrency has practical
applications in Go. If you find this package useful, please tell me about it!

## Examples

Here's some example code that demonstrates how to perform common operations:

```go
// create a shared variable
n := stm.NewVar(3)

// read a variable
var v int
stm.Atomically(func(tx *stm.Tx) {
	v = tx.Get(n).(int)
})
// or:
v = stm.AtomicGet(n).(int)

// write to a variable
stm.Atomically(func(tx *stm.Tx) {
	tx.Set(n, 12)
})
// or:
stm.AtomicSet(n, 12)

// update a variable
stm.Atomically(func(tx *stm.Tx) {
	cur := tx.Get(n).(int)
	tx.Set(n, cur-1)
})

// block until a condition is met
stm.Atomically(func(tx *stm.Tx) {
	cur := tx.Get(n).(int)
	if cur != 0 {
		tx.Retry()
	}
	tx.Set(n, 10)
})
// or:
stm.Atomically(func(tx *stm.Tx) {
	cur := tx.Get(n).(int)
	tx.Assert(cur == 0)
	tx.Set(n, 10)
})

// select among multiple (potentially blocking) transactions
stm.Atomically(stm.Select(
	// this function blocks forever, so it will be skipped
	func(tx *stm.Tx) { tx.Retry() },

	// this function will always succeed without blocking
	func(tx *stm.Tx) { tx.Set(n, 10) },

	// this function will never run, because the previous
	// function succeeded
	func(tx *stm.Tx) { tx.Retry() },
))
```

See [example_santa_test.go](example_santa_test.go) for a more complex example.

## Benchmarks

In synthetic benchmarks, STM seems to have a 1-5x performance penalty compared
to traditional mutex- or channel-based concurrency. However, note that these
benchmarks exhibit a lot of data contention, which is where STM is weakest.
For example, in `BenchmarkIncrementSTM`, each increment transaction retries an
average of 2.5 times. Less contentious benchmarks are forthcoming.

```
BenchmarkAtomicGet-4       	50000000	      26.7 ns/op
BenchmarkAtomicSet-4       	20000000	      65.7 ns/op

BenchmarkIncrementSTM-4    	     500	   2852492 ns/op
BenchmarkIncrementMutex-4  	    2000	    645122 ns/op
BenchmarkIncrementChannel-4	    2000	    986317 ns/op

BenchmarkReadVarSTM-4      	    5000	    268726 ns/op
BenchmarkReadVarMutex-4    	   10000	    248479 ns/op
BenchmarkReadVarChannel-4  	   10000	    240086 ns/op

```