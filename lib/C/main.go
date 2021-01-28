package libgopar_C

import (
	"C"
	"unsafe"
	"github.com/akalin/gopar/cmd/par"
)

//export gopar
func gopar(argc_ C.int, argv_ **C.char) int {
	defer func() {
		if r := recover(); r != nil {
			// libgopar only panics with strings currently
			fmt.Println("libgopar panicked! ", r)
			return 7
		}
	}()

	length := int(argc_)
	cStrings := (*[1 << 28]*C.char)(unsafe.Pointer(argv_))[:length:length]

	args_ := make([]string, length+1)
	args_[0] = "libgopar"
	for i, cString := range cStrings {
		args_[i+1] = C.GoString(cString)
	}
	return Par(args_)
}
