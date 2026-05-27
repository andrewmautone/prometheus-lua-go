package main

import (
	"fmt"
	"log"

	prometheus "github.com/andrewmautone/prometheus-lua-go"
)

func main() {
	o := prometheus.New(prometheus.Config{
		Preset: prometheus.PresetMedium,
		Seed:   42,
	})

	source := `
local function greet(name)
    return "Hello, " .. name .. "!"
end
print(greet("World"))
`

	out, err := o.Obfuscate(source)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
}
