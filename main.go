package main

import (
	"flag"
	"fmt"
	"github.com/thebagchi/ssh-utils/pkg"
	"reflect"
	"unsafe"
)

func FastStringToBytes(data string) []byte {
	hdr := *(*reflect.StringHeader)(unsafe.Pointer(&data))
	return *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: hdr.Data,
		Len:  hdr.Len,
		Cap:  hdr.Len,
	}))
}

func FastBytesToString(data []byte) string {
	hdr := *(*reflect.SliceHeader)(unsafe.Pointer(&data))
	return *(*string)(unsafe.Pointer(&reflect.StringHeader{
		Data: hdr.Data,
		Len:  hdr.Len,
	}))
}

func main() {
	var (
		username = flag.String("user", "", "username")
		password = flag.String("password", "", "password")
	)
	flag.Parse()
	if len(*username) == 0 || len(*password) == 0 {
		fmt.Println("Error: username or password cannot be empty")
		return
	}
	err := pkg.Upload("./main.go", "main.go", "localhost:22", *username, *password)
	if nil != err {
		fmt.Println("Error: ", err)
	}
	buffer, err := pkg.RunCommand("localhost:22", *username, *password, "pwd")
	if nil != err {
		fmt.Println("Error: ", err)
	} else {
		fmt.Println("Output: ", FastBytesToString(buffer))
	}

}
