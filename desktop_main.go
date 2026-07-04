//go:build desktop

package main

import (
	"embed"
	"log"

	"cursorbridge/internal/app"
	"cursorbridge/internal/desktop"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	applicationService, err := app.New(app.Options{
		ProxyAddr: "127.0.0.1:18080",
	})
	if err != nil {
		log.Fatal(err)
	}

	wailsApp := application.New(application.Options{
		Name:        "Cursor助手",
		Description: "Local Cursor IDE proxy and BYOK model gateway",
		Services: []application.Service{
			application.NewService(desktop.NewService(applicationService)),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Cursor助手",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 44,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(238, 242, 245),
		URL:              "/",
		Width:            1180,
		Height:           820,
		MinWidth:         860,
		MinHeight:        620,
	})

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}
