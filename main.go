package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	fmt.Println("go-wasm")
	err := os.MkdirAll("/tmp", 0755)
	if err != nil {
		fmt.Println("err", err)
		return
	}
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
