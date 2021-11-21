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
	log "github.com/sirupsen/logrus"
)

func Render(cfg *config.EdgeConfigSchema) {

	appCfg = cfg

	myApp := app.New()
	myWindow := myApp.NewWindow(cfg.ControllerName)

	// main content
	tabs := container.NewAppTabs(
		getMessagesTab(),
		getCredentialsTab(),
		getBalancesTab(),
		getDashboardTab(),
		getMarketPlaceTab(),
		getLogTab(),
	)

	myWindow.SetContent(
		container.NewMax(
			tabs,
			//footer,
		),
	)

	myWindow.SetOnClosed(func() {
		log.Infoln(">>>>>>> TERMINATING <<<<<<<")
	})

	// run the dispatcher that updates the ui
	go dispatcher(cfg.RuntimeMsgs.Notification)
	// lanuch the app
	myWindow.Resize(fyne.NewSize(940, 660))
	myWindow.ShowAndRun()
}

func getMessagesTab() *container.TabItem {

	list := widget.NewListWithData(
		contacts,
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(di binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(di.(binding.String))
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

	// footer stuff
	rightPanel := container.NewBorder(
		nil,
		container.NewBorder(
			nil,
			nil,
			nil,
			widget.NewButtonWithIcon("", theme.MailSendIcon(), executeCmd),
			widget.NewEntryWithData(userCommand),
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

	// public credentials
	pubList := widget.NewListWithData(
		publicCredentials,
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(di binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(di.(binding.String))
		},
	)
	pubList.OnSelected = publicCredentialSelected
	// private credentials
	privateList := widget.NewListWithData(
		privateCredentials,
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(di binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(di.(binding.String))
		},
	)
	privateList.OnSelected = privateCredentialSelected
	//
	leftPanel := widget.NewAccordion(
		widget.NewAccordionItem("Private Credentials", privateList),
		widget.NewAccordionItem("Public Credentials", pubList),
	)

	// right panel
	msgPanel := widget.NewLabelWithData(credentialData)
	msgScroll := container.NewScroll(msgPanel)

	body := container.NewHSplit(leftPanel, msgScroll)
	main := container.New(layout.NewMaxLayout(), body)

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
	msgPanel := widget.NewLabelWithData(logData)
	main := container.NewScroll(msgPanel)
	return container.NewTabItem("Logs", main)
}
