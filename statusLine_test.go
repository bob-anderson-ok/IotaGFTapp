package main

import (
	"fyne.io/fyne/v2/app"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_statusLine(t *testing.T) {
	var appConfig Config
	app.New() // Necessary in order to create widgets
	sl := makeStatusLine(&appConfig)
	assert.Equal(t, 4, len(sl.Objects))
	assert.Equal(t, "Latitude: not available", appConfig.latitudeStatus.Text)
	assert.Equal(t, "Longitude: not available", appConfig.longitudeStatus.Text)
	assert.Equal(t, "Altitude: not available", appConfig.altitudeStatus.Text)
	assert.Equal(t, "UTC date/time: not available", appConfig.dateTimeStatus.Text)
}

//func examiner(t reflect.Type, depth int) {
//	fmt.Println(strings.Repeat("\t", depth), "Type is", t.Name(), "and kind is", t.Kind())
//	switch t.Kind() {
//	case reflect.Array, reflect.Chan, reflect.Map, reflect.Ptr, reflect.Slice:
//		fmt.Println(strings.Repeat("\t", depth+1), "Contained type:")
//		examiner(t.Elem(), depth+1)
//	case reflect.Struct:
//		for i := 0; i < t.NumField(); i++ {
//			f := t.Field(i)
//			if i == 3 {
//				fmt.Println("Field[3]:", t.Field(i))
//			}
//			fmt.Println(strings.Repeat("\t", depth+1), "Field", i+1, "name is", f.Name, "type is", f.Type.Name(), "and kind is", f.Type.Kind())
//			if f.Tag != "" {
//				fmt.Println(strings.Repeat("\t", depth+2), "Tag is", f.Tag)
//				fmt.Println(strings.Repeat("\t", depth+2), "tag1 is", f.Tag.Get("tag1"), "tag2 is", f.Tag.Get("tag2"))
//			}
//		}
//	default:
//		panic("unhandled default case")
//	}
//}
