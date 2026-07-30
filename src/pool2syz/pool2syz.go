package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Radon10043/cloud/src/pkg/pool"
)

var (
	flagPool         string
	flagValidComment bool
)

func main() {
	flag.StringVar(&flagPool, "pool", "", "Path to the pool file")
	flag.BoolVar(&flagValidComment, "valid-comment", false, "Include valid comments")
	flag.Parse()

	b, err := os.ReadFile(flagPool)
	if err != nil {
		panic(err)
	}
	po, err := pool.NewSpecPoolFromJson(string(b))
	if err != nil {
		panic(err)
	}
	syzl := po.Syzlang(
		pool.WithValidComment(flagValidComment),
	)
	fmt.Printf("%s\n", syzl)
}
