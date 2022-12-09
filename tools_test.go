package toolkit

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	var testTools Tools

	s := testTools.RandomString(10)
	if len(s) != 10 {
		t.Errorf("wrong length random string returned; expected 10, but got %d", len(s))
	}
}

var uploadTests = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{"allowed no rename", []string{"image/jpeg", "image/png"}, false, false},
	{"allowed rename", []string{"image/jpeg", "image/png"}, true, false},
	{"invalid file type", []string{"image/jpeg"}, false, true},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, e := range uploadTests {
		// set up a pipe to avoid buffering
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)

		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			defer func(writer *multipart.Writer) {
				err := writer.Close()
				if err != nil {

				}
			}(writer)
			defer wg.Done()

			// create the form data field 'file'
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error(err)
			}

			f, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error(err)
			}

			defer func(f *os.File) {
				err := f.Close()
				if err != nil {

				}
			}(f)

			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding the image")
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}

		}()

		// read from teh pip which receives data
		req := httptest.NewRequest(http.MethodPost, "/", pr)
		req.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedFileTypes = e.allowedTypes

		uploadedFiles, err := testTools.UploadFiles(req, "./testdata/uploads/", e.renameFile)
		if err != nil && !e.errorExpected {
			t.Error(err)
		}

		if !e.errorExpected {
			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)); err != nil {
				t.Errorf("%s: no file found; %s", e.name, err.Error())
			}

			_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName))
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: error expected but none recieved", e.name)
		}

		wg.Wait()

	}
}

func TestTools_UploadOneFile(t *testing.T) {
	// set up a pipe to avoid buffering
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer func(writer *multipart.Writer) {
			err := writer.Close()
			if err != nil {

			}
		}(writer)

		// create the form data field 'file'
		part, err := writer.CreateFormFile("file", "./testdata/img.png")
		if err != nil {
			t.Error(err)
		}

		f, err := os.Open("./testdata/img.png")
		if err != nil {
			t.Error(err)
		}

		defer func(f *os.File) {
			err := f.Close()
			if err != nil {

			}
		}(f)

		img, _, err := image.Decode(f)
		if err != nil {
			t.Error("error decoding the image")
		}

		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
		}

	}()

	// read from teh pip which receives data
	req := httptest.NewRequest(http.MethodPost, "/", pr)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	var testTools Tools

	uploadedFile, err := testTools.UploadOneFile(req, "./testdata/uploads/")
	if err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFile.NewFileName)); err != nil {
		t.Errorf("no file found; %s", err.Error())
	}

	// clean up
	_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFile.NewFileName))

}

func TestTools_CreateDirIfNotExist(t *testing.T) {
	var testTools Tools
	testDir := "./testdata/myDir"

	err := testTools.CreateDirIfNotExist(testDir)
	if err != nil {
		t.Error(err)
	}

	err = testTools.CreateDirIfNotExist(testDir)
	if err != nil {
		t.Error(err)
	}

	_ = os.Remove(testDir)

}

var slugTest = []struct {
	name          string
	s             string
	expected      string
	errorExpected bool
}{
	{"valid string", "now is the time", "now-is-the-time", false},
	{"complex string", "now is the time", "now-is-the-time", false},
	{"empty string", "NOW is the &*(^# time for & all & good &MEN + apple & poop 123", "now-is-the-time-for-all-good-men-apple-poop-123", false},
	{"japanese string", "こんにちは世界", "", true},
	{"japanese string and roman characrers", "hello こんにちは世界 world", "hello-world", false},
}

func TestTools_Slugify(t *testing.T) {
	var testTools Tools

	for _, e := range slugTest {
		slug, err := testTools.Slugify(e.s)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: error recieved when none expected: %s", e.name, err.Error())
		}

		if !e.errorExpected && slug != e.expected {
			t.Errorf("%s: wrong slug returned; expected %s, but got %s", e.name, e.expected, slug)
		}
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	var testTools Tools

	testTools.DownloadStaticFile(rr, r, "./testdata", "test.pdf", "passed.pdf")

	res := rr.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(res.Body)

	if res.Header["Content-Length"][0] != "2071792" {
		t.Error("wrong content length of", res.Header["Content-Length"][0])
	}

	if res.Header["Content-Disposition"][0] != "attachment; filename=\"passed.pdf\"" {
		t.Error("wrong content disposition")
	}

	_, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
}
