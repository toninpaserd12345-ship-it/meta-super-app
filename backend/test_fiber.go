package main

import (
	"fmt"
	"github.com/gofiber/fiber/v3"
)

func main() {
	var c fiber.Ctx
	fmt.Printf("%T\n", c)
}
