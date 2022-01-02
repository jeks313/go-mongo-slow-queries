package rest

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestResponseJSON(t *testing.T) {
	tests := []struct {
		Name           string
		Body           string
		ExpectedResult map[string]interface{}
		ExpectedError  error
	}{
		{
			Name:           "Json response parsing",
			Body:           "{\"key\": {\n\"key2\": \"value\"\n}}",
			ExpectedResult: map[string]interface{}{"key": map[string]interface{}{"key2": "value"}},
		},
		{
			Name:           "Invalid json response parsing error",
			Body:           "random text",
			ExpectedResult: map[string]interface{}{},
			ExpectedError:  &Error{},
		},
		{
			Name:           "Empty json response parsing",
			Body:           "",
			ExpectedResult: map[string]interface{}{},
		},
	}

	for i, test := range tests {
		Convey(fmt.Sprintf("Given the test case %d: %v", i, test.Name), t, func() {
			resp := &http.Response{
				Body:       ioutil.NopCloser(bytes.NewReader([]byte(test.Body))),
				StatusCode: 404,
			}
			result := make(map[string]interface{})

			err := ResponseJSON(resp, &result)

			if test.ExpectedError != nil {
				So(err, ShouldNotBeNil)
				So(err, ShouldHaveSameTypeAs, test.ExpectedError)
				So(err.Error(), ShouldContainSubstring, test.Body)
				So(err.Error(), ShouldContainSubstring, "failed to parse response")
				restErr, _ := err.(*Error)
				So(restErr.StatusCode, ShouldEqual, 404)
			} else {
				So(err, ShouldBeNil)
			}

			So(result, ShouldResemble, test.ExpectedResult)

		})
	}

}

func TestNilResultJSON(t *testing.T) {
	Convey("Given the test case result nil", t, func() {
		resp := &http.Response{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"key": "value"}`))),
		}
		err := ResponseJSON(resp, nil)

		So(err, ShouldBeNil)
	})

}
