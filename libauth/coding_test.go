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
func TestXEncode(t *testing.T) {
	Convey("XEncode should work", t, func() {
		So(QuirkBase64Encode(*XEncode("", "aa0edd0fff7dd9f1f0ae4e981ec0114c7b0bf6f67c4895bed4f4ac634e97ecf2")), ShouldEqual, "")
		So(QuirkBase64Encode(*XEncode("1", "aa0edd0fff7dd9f1f0ae4e981ec0114c7b0bf6f67c4895bed4f4ac634e97ecf2")), ShouldEqual, "NmsaR0fCm5H=")
		So(QuirkBase64Encode(*XEncode("agfawegwq12834eqrge", "aa0edd0fff7dd9f1f0ae4e981ec0114c7b0bf6f67c4895bed4f4ac634e97ecf2")), ShouldEqual, "DAxHygvRUjlDyJjmvChIzuavMsjy7B9L")
		So(QuirkBase64Encode(*XEncode("agfawegwq12834eqrge", "0000000000000000000000000000000000000000000000000000000000000000")), ShouldEqual, "TOdQ9ggF2y/mskS6Orkg+eUZIok9vqJr")
		So(QuirkBase64Encode(*XEncode("9$02%8r89)(&22{}we[f]|s", "aa0edd0fff7dd9f1f0ae4e981ec0114c7b0bf6f67c4895bed4f4ac634e97ecf2")), ShouldEqual, "kCG+xmvGAhCV717Y80Fk0o1YJ8SYvBdnUmQoqS==")
	})
}
