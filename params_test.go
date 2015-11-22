package jasco

import (
	"github.com/gocraft/web"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestPathParams(t *testing.T) {
	Convey("Given a PathParams", t, func() {
		p := PathParams{
			req: &web.Request{
				PathParams: map[string]string{
					"str":     "value",
					"int":     "10",
					"neg_int": "-10",
				},
			},
		}

		Convey("when getting an optional string value", func() {
			s := p.String("str", "default")

			Convey("it should return the stored value", func() {
				So(s, ShouldEqual, "value")
			})
		})

		Convey("when getting a nonexistent string value", func() {
			s := p.String("nonexistent_str", "default")

			Convey("it should return the default value", func() {
				So(s, ShouldEqual, "default")
			})
		})

		Convey("when getting a required string value", func() {
			s, err := p.RequiredString("str")

			Convey("it should succeed", func() {
				So(err, ShouldBeNil)
			})

			Convey("it should return the stored value", func() {
				So(s, ShouldEqual, "value")
			})
		})

		Convey("when getting a required but nonexistent string value", func() {
			_, err := p.RequiredString("nonexistent_str")

			Convey("it should fail", func() {
				So(err, ShouldNotBeNil)
			})
		})

		Convey("when getting an optional positive integer value", func() {
			i, err := p.Int("int", 20)

			Convey("it should succeed", func() {
				So(err, ShouldBeNil)
			})

			Convey("it should return the stored value", func() {
				So(i, ShouldEqual, 10)
			})
		})

		Convey("when getting a nonexistent positive integer value", func() {
			i, err := p.Int("nonexistent_int", 20)

			Convey("it should succeed", func() {
				So(err, ShouldBeNil)
			})

			Convey("it should return the default value", func() {
				So(i, ShouldEqual, 20)
			})
		})

		Convey("when getting a required positive integer value", func() {
			i, err := p.RequiredInt("int")

			Convey("it should succeed", func() {
				So(err, ShouldBeNil)
			})

			Convey("it should return the stored value", func() {
				So(i, ShouldEqual, 10)
			})
		})

		Convey("when getting a required but nonexistent positive integer value", func() {
			_, err := p.RequiredInt("nonexistent_int")

			Convey("it should fail", func() {
				So(err, ShouldNotBeNil)
			})
		})

		Convey("when getting a negative integer", func() {
			_, err := p.Int("neg_int", 20)

			Convey("it should fail", func() {
				So(err, ShouldNotBeNil)
			})
		})

		Convey("when getting a string as an integer", func() {
			_, err := p.Int("str", 10)

			Convey("it should fail", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}
