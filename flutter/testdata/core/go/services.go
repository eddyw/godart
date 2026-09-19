package app

import (
	"fmt"

	"example.dev/gocore/bindings/services"
	"gart.dev/ui"
	"example.dev/gocore/bindings/material"
)

func ServiceTree() ui.Node {
	var children []ui.Node
	for _, mode := range []string{"future", "error", "stream", "cancel", "stream error"} {
		children = append(children, material.TextButton{
			Child:     ui.Text{Data: mode},
			OnPressed: func() { go serviceProbe(mode) },
		})
	}
	children = append(children, ui.Watch{Build: func() ui.Node { return ui.Text{Data: root.shared.Get()} }})
	return ui.Column{Children: children}
}

func serviceProbe(mode string) {
	var result string
	var err error
	switch mode {
	case "future":
		result, err = services.Greeting("Go")
	case "error":
		result, err = services.Failure()
	case "stream", "cancel":
		stream, openErr := services.Numbers()
		err = openErr
		if err == nil {
			sum := 0
			for {
				value, ok, nextErr := stream.Next()
				err = nextErr
				if err != nil || !ok {
					break
				}
				sum += value
				if mode == "cancel" {
					err = stream.Cancel()
					break
				}
			}
			result = fmt.Sprint(sum)
		}
	case "stream error":
		stream, openErr := services.Broken()
		err = openErr
		if err == nil {
			_, _, err = stream.Next()
			if err == nil {
				_, _, err = stream.Next()
			}
		}
	}
	if err != nil {
		result = err.Error()
	}
	ui.Post(func() { root.shared.Set(result) })
}
