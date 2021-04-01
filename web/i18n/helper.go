// SPDX-License-Identifier: MIT

// Package i18n wraps around github.com/nicksnyder/go-i18n mostly so that we don't have to deal with i18n.LocalizeConfig struct literals everywhere.
package i18n

import (
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"go.mindeco.de/http/render"
	"golang.org/x/text/language"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
)

type Helper struct {
	bundle *i18n.Bundle
}

func New(r repo.Interface) (*Helper, error) {

	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	// parse toml files and add them to the bundle
	walkFn := func(path string, info os.FileInfo, rs io.Reader, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, "toml") {
			return nil
		}

		mfb, err := ioutil.ReadAll(rs)
		if err != nil {
			return err
		}
		_, err = bundle.ParseMessageFileBytes(mfb, path)
		if err != nil {
			return fmt.Errorf("i18n: failed to parse file %s: %w", path, err)
		}
		fmt.Println("loaded", path)
		return nil
	}

	// walk the embedded defaults
	err := fs.WalkDir(Defaults, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		r, err := Defaults.Open(path)
		if err != nil {
			return err
		}

		err = walkFn(path, info, r, err)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("i18n: failed to iterate localizations: %w", err)
	}

	// walk the local filesystem for overrides and additions
	err = filepath.Walk(r.GetPath("i18n"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		r, err := os.Open(path)
		if err != nil {
			return err
		}
		defer r.Close()

		err = walkFn(path, info, r, err)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("i18n: failed to iterate localizations: %w", err)
	}

	return &Helper{bundle: bundle}, nil
}

func (h Helper) GetRenderFuncs() []render.Option {
	var opts = []render.Option{
		render.InjectTemplateFunc("i18npl", func(r *http.Request) interface{} {
			loc := h.FromRequest(r)
			return loc.LocalizePlurals
		}),

		render.InjectTemplateFunc("i18n", func(r *http.Request) interface{} {
			loc := h.FromRequest(r)
			return loc.LocalizeSimple
		}),
	}
	return opts
}

type Localizer struct {
	loc *i18n.Localizer
}

func (h Helper) newLocalizer(lang string, accept ...string) *Localizer {
	var langs = []string{lang}
	langs = append(langs, accept...)
	var l Localizer
	l.loc = i18n.NewLocalizer(h.bundle, langs...)
	return &l
}

// FromRequest returns a new Localizer for the passed helper,
// using form value 'lang' and Accept-Language http header from the passed request.
// TODO: user settings/cookie values?
func (h Helper) FromRequest(r *http.Request) *Localizer {
	lang := r.FormValue("lang")
	accept := r.Header.Get("Accept-Language")
	return h.newLocalizer(lang, accept)
}

func (l Localizer) LocalizeSimple(messageID string) string {
	msg, err := l.loc.Localize(&i18n.LocalizeConfig{
		MessageID: messageID,
	})
	if err == nil {
		return msg
	}

	panic(fmt.Sprintf("i18n/error: failed to localize label %s: %s", messageID, err))
}

func (l Localizer) LocalizeWithData(messageID string, tplData map[string]string) string {
	msg, err := l.loc.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: tplData,
	})
	if err == nil {
		return msg
	}

	panic(fmt.Sprintf("i18n/error: failed to localize label %s: %s", messageID, err))
}

func (l Localizer) LocalizePlurals(messageID string, pluralCount int) string {
	msg, err := l.loc.Localize(&i18n.LocalizeConfig{
		MessageID:   messageID,
		PluralCount: pluralCount,
		TemplateData: map[string]int{
			"Count": pluralCount,
		},
	})
	if err == nil {
		return msg
	}

	panic(fmt.Sprintf("i18n/error: failed to localize label %s: %s", messageID, err))
}

func (l Localizer) LocalizePluralsWithData(messageID string, pluralCount int, tplData map[string]string) string {
	msg, err := l.loc.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		PluralCount:  pluralCount,
		TemplateData: tplData,
	})
	if err == nil {
		return msg
	}

	panic(fmt.Sprintf("i18n/error: failed to localize label %s: %s", messageID, err))
}
