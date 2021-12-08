package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/allinbits/cosmos-cash-agent/pkg/config"
	"github.com/allinbits/cosmos-cash-agent/pkg/model"
	log "github.com/sirupsen/logrus"
)

var (
	mainWindow fyne.Window
)

func Render(cfg *config.EdgeConfigSchema) {

	appCfg = cfg

	myApp := app.New()
	mainWindow = myApp.NewWindow(cfg.ControllerName)

	// main content
	tabs := container.NewAppTabs(
		getMessagesTab(),
		getCredentialsTab(),
		getBalancesTab(),
		getDashboardTab(),
		getMarketPlaceTab(),
		getLogTab(),
	)

	mainWindow.SetContent(
		container.NewMax(
			tabs,
			//footer,
		),
	)

	mainWindow.SetOnClosed(func() {
		log.Infoln(">>>>>>> TERMINATING <<<<<<<")
	})

	// run the dispatcher that updates the ui
	go dispatcher(mainWindow, cfg.RuntimeMsgs.Notification)
	// lanuch the app
	mainWindow.Resize(fyne.NewSize(940, 660))
	mainWindow.ShowAndRun()
}

func getMessagesTab() *container.TabItem {

	list := widget.NewListWithData(
		contacts,
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(di binding.DataItem, o fyne.CanvasObject) {
			u := di.(binding.Untyped)
			i, _ := u.Get()
			c := i.(model.Contact)
			o.(*widget.Label).SetText(c.TheirLabel)
		},
	)
	list.OnSelected = contactSelected

	// msg list
	msgList := widget.NewListWithData(
		messages,
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(di binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(di.(binding.String))
		},
	)

	//msgPanel := container.NewVBox()
	msgScroll := container.NewScroll(msgList)

	input := widget.NewEntryWithData(userCommand)
	input.OnSubmitted = func(_ string) {
		executeCmd()
	}

	// footer stuff
	rightPanel := container.NewBorder(
		nil,
		container.NewBorder(
			nil,
			nil,
			nil,
			widget.NewButtonWithIcon("", theme.MailSendIcon(), executeCmd),
			input,
		),
		nil,
		nil,
		msgScroll,
	)

	body := container.NewHSplit(list, rightPanel)
	main := container.New(layout.NewMaxLayout(), body)

	return container.NewTabItem("Messages", main)
}

func getDashboardTab() *container.TabItem {

	main := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Wallet owner: \n%s", appCfg.ControllerName)),
		widget.NewLabel(fmt.Sprintf("Wallet DID ID: \ndid:cosmos:net:%s:%s", appCfg.ChainID, appCfg.ControllerDidID)),
	)

	return container.NewTabItem("Dashboard", main)
}

func getCredentialsTab() *container.TabItem {

	var pubList, privList *widget.List

	// public credentials
	pubList = widget.NewListWithData(
		publicCredentials,
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(di binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(di.(binding.String))
		},
	)

	// private credentials
	privList = widget.NewListWithData(
		privateCredentials,
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(di binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(di.(binding.String))
		},
	)

	// selection actions
	pubList.OnSelected = func(id widget.ListItemID) {
		privList.UnselectAll()
		publicCredentialSelected(id)
	}

	privList.OnSelected = func(id widget.ListItemID) {
		pubList.UnselectAll()
		privateCredentialSelected(id)
	}

	leftPanel := container.NewVSplit(
		widget.NewCard("Private Credentials", "Off-chain", privList),
		widget.NewCard("Public Credentials", "On-chain", pubList),
	)

	// right panel

	msgPanel := widget.NewEntryWithData(credentialData)
	rightPanel := container.NewScroll(msgPanel)

	body := container.NewHSplit(leftPanel, rightPanel)
	main := container.NewMax(body)

	return container.NewTabItem("Credentials", main)
}

func getBalancesTab() *container.TabItem {

	list := widget.NewListWithData(
		balances,
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(di binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(di.(binding.String))
		},
	)

	list.OnSelected = balancesSelected

	msgPanel := widget.NewLabelWithData(balancesChainOfTrust)
	msgScroll := container.NewScroll(msgPanel)

	body := container.NewHSplit(list, msgScroll)
	main := container.New(layout.NewMaxLayout(), body)

	return container.NewTabItem("Balances", main)
}

// tabs for marketplace
func getMarketPlaceTab() *container.TabItem {

	list := widget.NewListWithData(
		marketplaces,
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(di binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(di.(binding.String))
		},
	)

	list.OnSelected = marketplacesSelected

	msgPanel := widget.NewLabelWithData(marketplaceData)
	msgScroll := container.NewScroll(msgPanel)

	body := container.NewHSplit(list, msgScroll)
	main := container.New(layout.NewMaxLayout(), body)

	return container.NewTabItem("Marketplace", main)
}

func getLogTab() *container.TabItem {
	list := widget.NewListWithData(
		logData,
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(di binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(di.(binding.String))
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		entry, _ := logData.GetValue(id)
		// copy to clipboard the name
		appCfg.RuntimeMsgs.Notification <- config.NewAppMsg(config.MsgClipboard, entry)
		list.UnselectAll()
	}
	main := container.NewMax(container.NewScroll(list))
	return container.NewTabItem("Logs", main)
}

