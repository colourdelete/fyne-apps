package main

import (
	"fyne.io/apps/pkg/apps"
	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"log"
)

func main() {
	app := app.New()
	win := app.NewWindow("Fyne Apps")

	data, err := apps.LoadAppListFromWeb()
	if err != nil {
		log.Println("Web failed, reading cache")
		data, err = apps.LoadAppListFromCache()
		if err != nil {
			fyne.LogError("Load error", err)
			return
		}
	}

	defer data.Close()

	appList, err := apps.ParseAppList(data)
	if err != nil {
		fyne.LogError("Parse error", err)
		return
	}

	win.SetContent(apps.NewApps(appList, win))
	win.Resize(fyne.NewSize(680, 400))

	win.ShowAndRun()
}
