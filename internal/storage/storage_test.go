package storage

import (
	"fmt"
	"log"
	"testing"
)

func TestStorage(t *testing.T) {
	get, err := Storage.Get("hello world")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(get)

	err = Storage.Set("hello world", []byte("你好世界"))
	if err != nil {
		log.Fatalln(err)
	}

	value, err := Storage.Get("hello world")
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(string(value))
}
