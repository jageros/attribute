package utils

import (
	"log"
)

func CatchPanic(f func()) (err interface{}) {
	defer func() {
		err = recover()
		if err != nil {
			log.Printf("Catch panic: %s", err)
		}
	}()

	f()
	return
}

func RunPanicless(f func()) (panicless bool) {
	defer func() {
		err := recover()
		panicless = err == nil
		if err != nil {
			log.Printf("Catch panic: %s", err)
		}
	}()

	f()
	return
}

func RepeatUntilPanicless(f func()) {
	for !RunPanicless(f) {
	}
}
