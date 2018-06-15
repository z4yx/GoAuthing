package libauth

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestQuirkBase64Encode(t *testing.T) {
	Convey("QuirkBase64Encode should work", t, func() {
		So(QuirkBase64Encode("1"), ShouldEqual, "9+==")
		So(QuirkBase64Encode("2"), ShouldEqual, "9S==")
		So(QuirkBase64Encode("34"), ShouldEqual, "9z+=")
		So(QuirkBase64Encode("567"), ShouldEqual, "0FZ7")
		So(QuirkBase64Encode("\x00"), ShouldEqual, "LL==")
		So(QuirkBase64Encode("\x00\x00"), ShouldEqual, "LLL=")
		So(QuirkBase64Encode("\xff\x00\x00"), ShouldEqual, "AvLL")
		So(QuirkBase64Encode("\x01"), ShouldEqual, "L+==")
		So(QuirkBase64Encode("\x01==!@#$%^&*()"), ShouldEqual, "LFt52HLkRourRX//8+==")
		So(QuirkBase64Encode("\x01aAbB_+=-\x11"), ShouldEqual, "LaiVZYRs8ztfP+==")
	})
}
