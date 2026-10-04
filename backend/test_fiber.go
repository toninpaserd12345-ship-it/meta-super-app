package main

import (
	"github.com/gofiber/fiber/v3"
	"fmt"
)

func main() {
	var c fiber.Ctx
	fmt.Printf("%T\n", c)
}
