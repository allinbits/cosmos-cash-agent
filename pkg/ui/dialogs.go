package ui

import (
	"fmt"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/allinbits/cosmos-cash-agent/pkg/model"
	log "github.com/sirupsen/logrus"
	"reflect"
	"strconv"
)

// RenderRequestConfirmation render request confirmations, only accept or deny are displayed
func RenderRequestConfirmation(title string, data model.PresentationRequest, onAccept func(pr model.PresentationRequest), onAbort func(pr model.PresentationRequest)) {
	vo := reflect.ValueOf(data)
	to := vo.Type()

	var popUp *widget.PopUp
	var fields []*widget.FormItem

	for i := 0; i < to.NumField(); i++ {
		f := to.Field(i)
		v := vo.Field(i)

		label, hasLabel := f.Tag.Lookup("cash_label")
		// only render fields that have a label
		if !hasLabel {
			continue
		}

		e := widget.NewEntry()
		e.SetText(fmt.Sprintf("%v", v))
		e.Disable()

		w := widget.NewFormItem(label, e)
		w.HintText = f.Tag.Get("cash_hint")
		fields = append(fields, w)
	}

	// render the form
	form := &widget.Form{
		Items: fields,
		OnSubmit: func() {
			// hide the popUp
			popUp.Hide()
			// execute the callback
			log.Debugln("form data", data)
			onAccept(data)
		},
		OnCancel: func() {
			popUp.Hide()
			onAbort(data)
		},
		SubmitText: "Confirm",
		CancelText: "Abort",
	}
	// create the popUp
	popUp = widget.NewModalPopUp(
		widget.NewCard(title, "", form),
		mainWindow.Canvas(),
	)
	popUp.Show()
}

// RenderPresentationRequest render a presentation request in the ui
func RenderPresentationRequest(title string, data model.PresentationRequest, onSubmit func(interface{})) {
	answers := make(map[string]binding.String)
	var popUp *widget.PopUp
	var fields []*widget.FormItem

	var ptr reflect.Value
	var vo reflect.Value

	vo = reflect.ValueOf(data)
	to := vo.Type()



	ptr = reflect.New(to)
	temp := ptr.Elem()
	temp.Set(vo)

	for i := 0; i < to.NumField(); i++ {
		f := to.Field(i)
		v := vo.Field(i)

		label, hasLabel := f.Tag.Lookup("cash_label")
		// only render fields that have a label
		if !hasLabel {
			continue
		}
		hint := f.Tag.Get("cash_hint")
		_, readOnly := f.Tag.Lookup("cash_ro")

		b := binding.NewString()
		e := widget.NewEntryWithData(b)

		b.Set(fmt.Sprintf("%v", v))
		if readOnly {
			e.Disable()
		}

		answers[f.Name] = b
		// form item
		w := widget.NewFormItem(label, e)
		w.HintText = hint
		// fields
		fields = append(fields, w)
	}
	// render the form
	var fieldValue reflect.Value
	form := &widget.Form{
		Items: fields,
		OnSubmit: func() {
			// hide the popUp
			popUp.Hide()
			for k, b := range answers {
				fieldValue = temp.FieldByName(k)
				// value from the input
				s, _ := b.Get()
				switch fieldValue.Kind() {
				case reflect.String:
					fieldValue.SetString(s)
				case reflect.Float64:
					pf, err := strconv.ParseFloat(s, 64)
					if err != nil {
						log.Errorln("cannot parse float %v", s)
					}
					fieldValue.SetFloat(pf)
				default:
					log.Errorln("cannot handle type %v for field %v", fieldValue.Kind(), k)
				}
			}
			// execute the callback
			log.Debugln("form data", temp)
			onSubmit(temp.Interface())
		},
		OnCancel:   func() { popUp.Hide() },
		SubmitText: "Submit",
		CancelText: "Cancel",
	}

	// create the popUp
	popUp = widget.NewModalPopUp(
		widget.NewCard(title, "", form),
		mainWindow.Canvas(),
	)
	popUp.Show()
}
