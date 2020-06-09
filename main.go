package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"time"

	"github.com/johnstarich/go-wasm/log"
)

func main() {
	go func() {
		time.Sleep(2 * time.Second)
		buf := make([]byte, 1000000)
		n := runtime.Stack(buf, true)
		log.Error("stack", string(buf[:n]))
	}()
	fmt.Println("go-wasm")
	info, err := os.Stat(".")
	fmt.Println("info", info)
	if err != nil {
		fmt.Println("err", err)
		return
	}

	wd, err := os.Getwd()
	fmt.Println("wd", wd, err)

	fmt.Println("Writing 'hi'")
	err = ioutil.WriteFile("/hi", []byte("hi"), 0600)
	if err != nil {
		fmt.Println("write err", err)
		return
	}
	buf, err := ioutil.ReadFile("/hi")
	if err != nil {
		fmt.Println("read err", err)
		return
	}
	fmt.Println("read result: ", string(buf))
	fmt.Println("success!")
}
