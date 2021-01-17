package apps

import (
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/cmd/fyne/commands"
	"fyne.io/fyne/container"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"
)

type Apps struct {
	shownID, shownPkg, shownIcon string
	name, summary, date          *widget.Label
	developer, version           *widget.Label
	link                         *widget.Hyperlink
	icon, screenshot             *canvas.Image
}

func (w *Apps) loadAppDetail(app App) {
	w.shownID = app.ID
	w.shownPkg = app.Source.Package
	w.shownIcon = app.Icon

	w.name.SetText(app.Name)
	w.developer.SetText(app.Developer)
	w.version.SetText(app.Version)
	w.date.SetText(app.Date.Format("02 Jan 2006"))
	w.summary.SetText(app.Summary)

	w.icon.Resource = nil
	go setImageFromURL(w.icon, app.Icon)

	w.screenshot.Resource = nil
	if len(app.Screenshots) > 0 {
		go setImageFromURL(w.screenshot, app.Screenshots[0].Image)
	}
	w.screenshot.Refresh()

	parsed, err := url.Parse(app.Website)
	if err != nil {
		w.link.SetText("")
		return
	}
	w.link.SetText(parsed.Host)
	w.link.SetURL(parsed)
}

func setImageFromURL(img *canvas.Image, location string) {
	if location == "" {
		return
	}

	res, err := loadResourceFromURL(location)
	if err != nil {
		img.Resource = theme.WarningIcon()
	} else {
		img.Resource = res
	}

	canvas.Refresh(img)
}

func loadResourceFromURL(URL string) (fyne.Resource, error) {
	client := http.Client{
		Timeout: 1 * time.Second,
	}

	req, err := client.Get(URL)

	if err != nil {
		return nil, err
	}

	defer req.Body.Close()

	bytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	parsed, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}

	name := filepath.Base(parsed.Path)

	return fyne.NewStaticResource(name, bytes), nil
}

// iconHoverLayout specifies a layout that floats an icon image top right over other content.
type iconHoverLayout struct {
	content, icon fyne.CanvasObject
}

func (i *iconHoverLayout) Layout(_ []fyne.CanvasObject, size fyne.Size) {
	i.content.Resize(size)

	i.icon.Resize(fyne.NewSize(64, 64))
	i.icon.Move(fyne.NewPos(size.Width-i.icon.Size().Width, 0))
}

func (i *iconHoverLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return i.content.MinSize()
}

func (w *Apps) installer(win fyne.Window, progBar *widget.ProgressBarInfinite, progLabel *widget.Label) func() {
	return func() {
		shownPkg := w.shownPkg
		shownIcon := w.shownIcon
		if shownPkg == "fyne.io/apps" {
			dialog.ShowInformation("System app", "Cannot overwrite the installer app", win)
			return
		}

		progLabel.SetText(fmt.Sprintf("Installing %s...", shownPkg))
		progBar.Start()
		defer progBar.Stop()

		tmpIconChan := make(chan string)
		go func() {
			tmpIconChan <- downloadIcon(shownIcon)
		}()

		tmpIcon := <-tmpIconChan
		get := commands.NewGetter()
		get.SetIcon(tmpIcon)
		err := get.Get(shownPkg)

		if err != nil {
			progLabel.SetText(err.Error())
			dialog.ShowError(err, win)
		} else {
			progLabel.SetText(fmt.Sprintf("Installed %s.", shownPkg))
		}

		err = os.Remove(tmpIcon)
		if err != nil {
			dialog.ShowError(err, win)
		}
	}
}

func NewApps(apps AppList, win fyne.Window) fyne.CanvasObject {
	w := &Apps{}
	w.name = widget.NewLabel("")
	w.developer = widget.NewLabel("")
	w.link = widget.NewHyperlink("", nil)
	w.summary = widget.NewLabel("")
	w.summary.Wrapping = fyne.TextWrapWord
	w.version = widget.NewLabel("")
	w.date = widget.NewLabel("")
	w.icon = &canvas.Image{}
	w.icon.FillMode = canvas.ImageFillContain
	w.screenshot = &canvas.Image{}
	w.screenshot.SetMinSize(fyne.NewSize(320, 240))
	w.screenshot.FillMode = canvas.ImageFillContain

	dateAndVersion := fyne.NewContainerWithLayout(layout.NewGridLayout(2), w.date,
		widget.NewForm(&widget.FormItem{Text: "Version", Widget: w.version}))

	form := widget.NewForm(
		&widget.FormItem{Text: "Name", Widget: w.name},
		&widget.FormItem{Text: "Developer", Widget: w.developer},
		&widget.FormItem{Text: "Website", Widget: w.link},
		&widget.FormItem{Text: "Summary", Widget: w.summary},
		&widget.FormItem{Text: "Date", Widget: dateAndVersion},
	)

	details := fyne.NewContainerWithLayout(&iconHoverLayout{content: form, icon: w.icon}, form, w.icon)

	list := widget.NewList(
		func() int {
			return len(apps)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("A longish app name")
		},
		func(id int, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(apps[id].Name)
		},
	)
	list.OnSelected = func(id int) {
		w.loadAppDetail(apps[id])
	}

	progBar := widget.NewProgressBarInfinite()
	progBar.Stop() // TODO: allow loading other apps while installing

	progLabel := widget.NewLabel("Loading...")
	buttons := container.NewHBox(
		layout.NewSpacer(),
		progBar,
		progLabel,
		widget.NewButton("Install", w.installer(win, progBar, progLabel)),
	)
	progLabel.SetText("Ready.")

	if len(apps) > 0 {
		w.loadAppDetail(apps[0])
	}
	content := container.NewBorder(details, nil, nil, nil, w.screenshot)
	return container.NewBorder(
		nil,
		nil,
		list,
		nil,
		container.NewBorder(nil, buttons, nil, nil, content),
	)
}

func downloadIcon(url string) string {
	req, err := http.Get(url)
	if err != nil {
		fyne.LogError("Failed to access icon url: "+url, err)
		return ""
	}

	tmp, err := ioutil.TempFile(os.TempDir(), "fyne-icon-*.png")
	if err != nil {
		fyne.LogError("Failed to create temporary file", err)
		return ""
	}
	defer tmp.Close()

	data, err := ioutil.ReadAll(req.Body)

	if err != nil {
		fyne.LogError("Failed tread icon data", err)
		return ""
	}

	_, err = tmp.Write(data)
	if err != nil {
		fyne.LogError("Failed to get write icon to: "+tmp.Name(), err)
		return ""
	}

	return tmp.Name()
}
