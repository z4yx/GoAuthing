package libauth

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestExtractJSONFromJSONP(t *testing.T) {

	Convey("Extracting JSON from JSONP...", t, func() {
		json, err := extractJSONFromJSONP("cb()", "cb")
		So(err, ShouldBeNil)
		So(json, ShouldEqual, "")
		json, err = extractJSONFromJSONP("C({})", "C")
		So(err, ShouldBeNil)
		So(json, ShouldEqual, "{}")
		json, err = extractJSONFromJSONP(`jQuery({"key1": 1234})`, "jQuery")
		So(err, ShouldBeNil)
		So(json, ShouldEqual, `{"key1": 1234}`)
		json, err = extractJSONFromJSONP("C({})", "")
		So(err, ShouldNotBeNil)
		json, err = extractJSONFromJSONP("C({})", "Q")
		So(err, ShouldNotBeNil)
		json, err = extractJSONFromJSONP("C({}", "C")
		So(err, ShouldNotBeNil)
		json, err = extractJSONFromJSONP("", "C")
		So(err, ShouldNotBeNil)
	})
}

// func TestBuildLoginParams(t *testing.T) {
// 	loggo.ConfigureLoggers("libauth=DEBUG")
// 	buildLoginParams("hello", "pass", "32f23b9c2229fd034f6d5160d8b4536496af550efc45113635689d2d8f12ffad")
// }
