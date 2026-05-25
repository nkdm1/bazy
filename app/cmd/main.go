package main

import (
	"github.com/nkdm1/bazy/internal/api"
)

func main() {
	api := api.Init()
	api.Run(api.Mount())
}
