package main

import (
	"C"
	"unsafe"
	"fmt"

	"github.com/akalin/gopar/libgopar"
)

//export gopar
func gopar(argc_ C.int, argv_ **C.char) int {
	retval := 7
	defer func() {
		if r := recover(); r != nil {
			// libgopar only panics with strings currently
			fmt.Println("libgopar panicked! ", r)
		}
	}()

	length := int(argc_)
	cStrings := (*[1 << 28]*C.char)(unsafe.Pointer(argv_))[:length:length]

	args_ := make([]string, length+1)
	args_[0] = "libgopar"
	for i, cString := range cStrings {
		args_[i+1] = C.GoString(cString)
	}
	retval = libgopar.Par(args_,true)
	return retval
}

func main(){}
