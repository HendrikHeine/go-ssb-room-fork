package admin

import (
	"bytes"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/webassert"
	refs "go.mindeco.de/ssb-refs"
)

func TestAliasesOverview(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	lst := []roomdb.Alias{
		{ID: 1, Name: "alice", Feed: refs.FeedRef{ID: bytes.Repeat([]byte{0}, 32), Algo: "fake"}},
		{ID: 2, Name: "bob", Feed: refs.FeedRef{ID: bytes.Repeat([]byte("1312"), 8), Algo: "test"}},
		{ID: 3, Name: "cleo", Feed: refs.FeedRef{ID: bytes.Repeat([]byte("acab"), 8), Algo: "true"}},
	}
	ts.AliasesDB.ListReturns(lst, nil)

	overviewURL := ts.URLTo(router.AdminAliasesOverview)

	html, resp := ts.Client.GetHTML(overviewURL)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminAliasesWelcome"},
		{"title", "AdminAliasesTitle"},
		{"#aliasCount", "ListCountPlural"},
	})

	a.EqualValues(html.Find("#theList li").Length(), 3)

	lst = []roomdb.Alias{
		{ID: 666, Name: "dave", Feed: refs.FeedRef{ID: bytes.Repeat([]byte{1}, 32), Algo: "one"}},
	}
	ts.AliasesDB.ListReturns(lst, nil)

	html, resp = ts.Client.GetHTML(overviewURL)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminAliasesWelcome"},
		{"title", "AdminAliasesTitle"},
		{"#aliasCount", "ListCountSingular"},
	})

	elems := html.Find("#theList li")
	a.EqualValues(elems.Length(), 1)

	// check for link to Revoke confirm link
	link, yes := elems.ContentsFiltered("a").Attr("href")
	a.True(yes, "a-tag has href attribute")
	a.Equal("/admin/aliases/revoke/confirm?id=666", link)
}

func TestAliasesRevokeConfirmation(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	testKey, err := refs.ParseFeedRef("@x7iOLUcq3o+sjGeAnipvWeGzfuYgrXl8L4LYlxIhwDc=.ed25519")
	a.NoError(err)
	testEntry := roomdb.Alias{ID: 666, Name: "the-test-name", Feed: *testKey}
	ts.AliasesDB.GetByIDReturns(testEntry, nil)

	urlRevokeConfirm := ts.URLTo(router.AdminAliasesRevokeConfirm, "id", 3)

	html, resp := ts.Client.GetHTML(urlRevokeConfirm)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	a.Equal(testKey.Ref(), html.Find("pre#verify").Text(), "has the key for verification")

	form := html.Find("form#confirm")

	method, ok := form.Attr("method")
	a.True(ok, "form has method set")
	a.Equal("POST", method)

	action, ok := form.Attr("action")
	a.True(ok, "form has action set")

	addURL := ts.URLTo(router.AdminAliasesRevoke)
	a.Equal(addURL.Path, action)

	webassert.ElementsInForm(t, form, []webassert.FormElement{
		{Name: "name", Type: "hidden", Value: testEntry.Name},
	})
}

func TestAliasesRevoke(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	urlRevoke := ts.URLTo(router.AdminAliasesRevoke)
	overviewURL := ts.URLTo(router.AdminAliasesOverview)

	ts.AliasesDB.RevokeReturns(nil)

	addVals := url.Values{"name": []string{"the-name"}}
	rec := ts.Client.PostForm(urlRevoke, addVals)
	a.Equal(http.StatusTemporaryRedirect, rec.Code)
	a.Equal(overviewURL.Path, rec.Header().Get("Location"))
	a.True(len(rec.Result().Cookies()) > 0, "got a cookie")

	// check flash messages
	doc, resp := ts.Client.GetHTML(overviewURL)
	a.Equal(http.StatusOK, resp.Code)
	flashes := doc.Find("#flashes-list").Children()
	a.Equal(1, flashes.Length())
	a.Equal("AdminAliasRevoked", flashes.Text())

	a.Equal(1, ts.AliasesDB.RevokeCallCount())
	_, theName := ts.AliasesDB.RevokeArgsForCall(0)
	a.EqualValues("the-name", theName)

	// now for unknown ID
	ts.AliasesDB.RevokeReturns(roomdb.ErrNotFound)
	addVals = url.Values{"name": []string{"nope"}}
	rec = ts.Client.PostForm(urlRevoke, addVals)
	a.Equal(http.StatusTemporaryRedirect, rec.Code)
	a.Equal(overviewURL.Path, rec.Header().Get("Location"))
	a.True(len(rec.Result().Cookies()) > 0, "got a cookie")

	// check flash messages
	doc, resp = ts.Client.GetHTML(overviewURL)
	a.Equal(http.StatusOK, resp.Code)
	flashes = doc.Find("#flashes-list").Children()
	a.Equal(1, flashes.Length())
	a.Equal("ErrorNotFound", flashes.Text())
}
