package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type Task struct {
	ID         string            `json:"id"`
	Status     string            `json:"status"`
	Errors     map[string]string `json:"errors"`
	Urls       []string          `json:"urls"`
	ArchiveURL string            `json:"archive_url"`
}

func main() {
	a := app.New()
	w := a.NewWindow("LinkZipper GUI")
	serverHostPref := a.Preferences().StringWithFallback("serverHost", "localhost")
	useHTTPS := a.Preferences().BoolWithFallback("useHTTPS", false)
	serverHost := serverHostPref
	if strings.HasPrefix(serverHostPref, "http://") {
		serverHost = strings.TrimPrefix(serverHostPref, "http://")
		useHTTPS = false
	} else if strings.HasPrefix(serverHostPref, "https://") {
		serverHost = strings.TrimPrefix(serverHostPref, "https://")
		useHTTPS = true
	}
	serverPort := a.Preferences().StringWithFallback("serverPort", "8080")
	scheme := "http"
	if useHTTPS {
		scheme = "https"
	}
	serverURL := fmt.Sprintf("%s://%s:%s", scheme, serverHost, serverPort)

	var tasks []Task
	selected := -1
	var errorKeys []string

	statusLabel := widget.NewLabel("-")
	linksList := widget.NewList(
		func() int {
			if selected < 0 || selected >= len(tasks) {
				return 0
			}
			return len(tasks[selected].Urls)
		},
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if selected < 0 || selected >= len(tasks) {
				return
			}
			o.(*widget.Label).SetText(tasks[selected].Urls[i])
		},
	)
	errorList := widget.NewList(
		func() int { return len(errorKeys) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if selected < 0 || selected >= len(tasks) {
				return
			}
			k := errorKeys[i]
			o.(*widget.Label).SetText(fmt.Sprintf("%s: %s", k, tasks[selected].Errors[k]))
		},
	)

	linkEntry := widget.NewEntry()
	linkEntry.SetPlaceHolder("https://example.com/file.pdf")

	var taskList *widget.List
	var updateDetails func()
	var refreshStatus func()
	refreshTasks := func() {
		resp, err := http.Get(serverURL + "/tasks/list")
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			dialog.ShowError(fmt.Errorf("list tasks: %s", resp.Status), w)
			return
		}
		var data []Task
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			dialog.ShowError(err, w)
			return
		}
		tasks = data
		taskList.Refresh()
		if selected >= len(tasks) {
			selected = -1
		}
		updateDetails()
	}

	updateDetails = func() {
		if selected < 0 || selected >= len(tasks) {
			statusLabel.SetText("-")
			errorKeys = nil
			linksList.Refresh()
			errorList.Refresh()
			return
		}
		t := tasks[selected]
		statusLabel.SetText(t.Status)
		errorKeys = errorKeys[:0]
		for k := range t.Errors {
			errorKeys = append(errorKeys, k)
		}
		sort.Strings(errorKeys)
		linksList.Refresh()
		errorList.Refresh()
	}

	taskList = widget.NewList(
		func() int { return len(tasks) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			t := tasks[i]
			o.(*widget.Label).SetText(fmt.Sprintf("%s - %s", t.ID, t.Status))
		},
	)

	taskList.OnSelected = func(id widget.ListItemID) {
		selected = int(id)
		updateDetails()
	}

	createTask := func() {
		resp, err := http.Post(serverURL+"/tasks", "application/json", nil)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			dialog.ShowError(fmt.Errorf("create task: %s", resp.Status), w)
			return
		}
		var r struct {
			TaskID string `json:"task_id"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			dialog.ShowError(err, w)
			return
		}
		refreshTasks()
		for i, t := range tasks {
			if t.ID == r.TaskID {
				selected = i
				taskList.Select(i)
				break
			}
		}
	}

	addLink := func() {
		if selected < 0 {
			return
		}
		payload := map[string]string{
			"task_id": tasks[selected].ID,
			"url":     linkEntry.Text,
		}
		body, _ := json.Marshal(payload)
		resp, err := http.Post(serverURL+"/tasks/links", "application/json", bytes.NewReader(body))
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			dialog.ShowError(fmt.Errorf("add link: %s", string(b)), w)
			return
		}
		tasks[selected].Urls = append(tasks[selected].Urls, linkEntry.Text)
		linksList.Refresh()
		linkEntry.SetText("")
	}

	forceZip := func() {
		if selected < 0 {
			return
		}
		payload := map[string]string{"task_id": tasks[selected].ID}
		body, _ := json.Marshal(payload)
		resp, err := http.Post(serverURL+"/tasks/zip", "application/json", bytes.NewReader(body))
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			dialog.ShowError(fmt.Errorf("force zip: %s", string(b)), w)
			return
		}
	}

	refreshStatus = func() {
		if selected < 0 {
			return
		}
		id := tasks[selected].ID
		resp, err := http.Get(serverURL + "/tasks/status/" + id)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			dialog.ShowError(fmt.Errorf("status: %s", resp.Status), w)
			return
		}
		var data struct {
			Status     string            `json:"status"`
			Errors     map[string]string `json:"errors"`
			ArchiveURL string            `json:"archive_url"`
			Urls       []string          `json:"urls"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			dialog.ShowError(err, w)
			return
		}
		tasks[selected].Status = data.Status
		tasks[selected].Errors = data.Errors
		tasks[selected].ArchiveURL = data.ArchiveURL
		tasks[selected].Urls = data.Urls
		updateDetails()
		taskList.RefreshItem(selected)
	}

	downloadTask := func() {
		if selected < 0 {
			return
		}
		if tasks[selected].ArchiveURL == "" {
			refreshStatus()
			if tasks[selected].ArchiveURL == "" {
				dialog.ShowInformation("Download", "Archive not ready", w)
				return
			}
		}
		save := dialog.NewFileSave(func(uc fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if uc == nil {
				return
			}
			resp, err := http.Get(serverURL + tasks[selected].ArchiveURL)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				dialog.ShowError(fmt.Errorf("download: %s", resp.Status), w)
				return
			}
			if _, err := io.Copy(uc, resp.Body); err != nil {
				dialog.ShowError(err, w)
			}
			uc.Close()
		}, w)
		save.SetFileName(tasks[selected].ID + ".zip")
		save.Show()
	}

	deleteTask := func() {
		if selected < 0 {
			return
		}
		id := tasks[selected].ID
		req, _ := http.NewRequest(http.MethodDelete, serverURL+"/tasks/delete/"+id, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		resp.Body.Close()
		refreshTasks()
	}

	settingsItem := fyne.NewMenuItem("Server", func() {
		hostEntry := widget.NewEntry()
		hostEntry.SetText(serverHost)
		portEntry := widget.NewEntry()
		portEntry.SetText(serverPort)
		httpsCheck := widget.NewCheck("", nil)
		httpsCheck.SetChecked(useHTTPS)
		dialog.ShowForm("Server Settings", "Save", "Cancel", []*widget.FormItem{
			widget.NewFormItem("Host", hostEntry),
			widget.NewFormItem("Port", portEntry),
			widget.NewFormItem("Use HTTPS", httpsCheck),
		}, func(ok bool) {
			if !ok {
				return
			}
			serverHost = hostEntry.Text
			serverPort = portEntry.Text
			useHTTPS = httpsCheck.Checked
			scheme := "http"
			if useHTTPS {
				scheme = "https"
			}
			serverURL = fmt.Sprintf("%s://%s:%s", scheme, serverHost, serverPort)
			a.Preferences().SetString("serverHost", serverHost)
			a.Preferences().SetString("serverPort", serverPort)
			a.Preferences().SetBool("useHTTPS", useHTTPS)
			refreshTasks()
		}, w)
	})
	w.SetMainMenu(fyne.NewMainMenu(fyne.NewMenu("Settings", settingsItem)))

	buttons := container.NewVBox(
		widget.NewButton("Create Task", createTask),
		widget.NewButton("Add Link", addLink),
		widget.NewButton("Refresh Status", refreshStatus),
		widget.NewButton("Force Zip", forceZip),
		widget.NewButton("Download", downloadTask),
		widget.NewButton("Delete Task", deleteTask),
	)

	statusCard := widget.NewCard("Status", "", statusLabel)
	linksCard := widget.NewCard("Links", "", container.NewMax(linksList))
	errorsCard := widget.NewCard("Errors", "", container.NewMax(errorList))
	right := container.NewVBox(statusCard, linksCard, errorsCard, linkEntry, buttons)
	left := container.NewMax(taskList)
	content := container.NewHSplit(left, container.NewMax(right))
	content.Offset = 0.3

	w.SetContent(content)
	w.Resize(fyne.NewSize(800, 400))
	refreshTasks()
	w.ShowAndRun()
}