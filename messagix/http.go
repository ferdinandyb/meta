package messagix

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"go.mau.fi/mautrix-meta/messagix/cookies"
	"go.mau.fi/mautrix-meta/messagix/types"
)

type HttpQuery struct {
	AcceptOnlyEssential  string `url:"accept_only_essential,omitempty"`
	Av                   string `url:"av,omitempty"`          // not required
	User                 string `url:"__user,omitempty"`      // not required
	A                    string `url:"__a,omitempty"`         // 1 or 0 wether to include "suggestion_keys" or not in the response - no idea what this is
	Req                  string `url:"__req,omitempty"`       // not required
	Hs                   string `url:"__hs,omitempty"`        // not required
	Dpr                  string `url:"dpr,omitempty"`         // not required
	Ccg                  string `url:"__ccg,omitempty"`       // not required
	Rev                  string `url:"__rev,omitempty"`       // not required
	S                    string `url:"__s,omitempty"`         // not required
	Hsi                  string `url:"__hsi,omitempty"`       // not required
	Dyn                  string `url:"__dyn,omitempty"`       // not required
	Csr                  string `url:"__csr"`                 // not required
	CometReq             string `url:"__comet_req,omitempty"` // not required but idk what this is
	FbDtsg               string `url:"fb_dtsg,omitempty"`
	Jazoest              string `url:"jazoest,omitempty"`                  // not required
	Lsd                  string `url:"lsd,omitempty"`                      // required
	SpinR                string `url:"__spin_r,omitempty"`                 // not required
	SpinB                string `url:"__spin_b,omitempty"`                 // not required
	SpinT                string `url:"__spin_t,omitempty"`                 // not required
	FbAPICallerClass     string `url:"fb_api_caller_class,omitempty"`      // not required
	FbAPIReqFriendlyName string `url:"fb_api_req_friendly_name,omitempty"` // not required
	Variables            string `url:"variables,omitempty"`
	ServerTimestamps     string `url:"server_timestamps,omitempty"` // "true" or "false"
	DocID                string `url:"doc_id,omitempty"`
	D                    string `url:"__d,omitempty"` // for insta
}

func (c *Client) NewHttpQuery() *HttpQuery {
	c.graphQLRequests++
	siteConfig := c.configs.browserConfigTable.SiteData
	dpr := strconv.FormatFloat(siteConfig.Pr, 'g', 4, 64)
	query := &HttpQuery{
		User:     c.configs.browserConfigTable.CurrentUserInitialData.UserID,
		A:        "1",
		Req:      strconv.Itoa(c.graphQLRequests),
		Hs:       siteConfig.HasteSession,
		Dpr:      dpr,
		Ccg:      c.configs.browserConfigTable.WebConnectionClassServerGuess.ConnectionClass,
		Rev:      strconv.Itoa(siteConfig.SpinR),
		S:        c.configs.WebSessionId,
		Hsi:      siteConfig.Hsi,
		Dyn:      c.configs.Bitmap.CompressedStr,
		Csr:      c.configs.CsrBitmap.CompressedStr,
		CometReq: c.configs.CometReq,
		FbDtsg:   c.configs.browserConfigTable.DTSGInitData.Token,
		Jazoest:  c.configs.Jazoest,
		Lsd:      c.configs.LsdToken,
		SpinR:    strconv.Itoa(siteConfig.SpinR),
		SpinB:    siteConfig.SpinB,
		SpinT:    strconv.Itoa(siteConfig.SpinT),
	}
	if c.configs.browserConfigTable.CurrentUserInitialData.UserID != "0" {
		query.Av = c.configs.browserConfigTable.CurrentUserInitialData.UserID
	}
	if c.platform == types.Instagram {
		query.D = "www"
	}
	return query
}

func (c *Client) MakeRequest(url string, method string, headers http.Header, payload []byte, contentType types.ContentType) (*http.Response, []byte, error) {
	newRequest, err := http.NewRequest(method, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, nil, err
	}

	if contentType != types.NONE {
		headers.Set("content-type", string(contentType))
	}

	newRequest.Header = headers

	response, err := c.http.Do(newRequest)
	defer func() {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
	}()
	if err != nil {
		c.UpdateProxy(fmt.Sprintf("http request error: %v", err.Error()))
		return nil, nil, err
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, nil, err
	}

	return response, responseBody, nil
}

func (c *Client) buildHeaders(withCookies bool) http.Header {
	headers := http.Header{}
	headers.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	headers.Set("accept-language", "en-US,en;q=0.9")
	headers.Set("dpr", DPR)
	headers.Set("user-agent", UserAgent)
	headers.Set("sec-ch-ua", SecCHUserAgent)
	headers.Set("sec-ch-ua-platform", SecCHPlatform)
	headers.Set("sec-ch-prefers-color-scheme", SecCHPrefersColorScheme)
	headers.Set("sec-ch-ua-full-version-list", SecCHFullVersionList)
	headers.Set("sec-ch-ua-mobile", SecCHMobile)
	headers.Set("sec-ch-ua-model", SecCHModel)
	headers.Set("sec-ch-ua-platform-version", SecCHPlatformVersion)

	if c.platform.IsMessenger() {
		c.addFacebookHeaders(&headers)
	} else {
		c.addInstagramHeaders(&headers)
	}

	if c.cookies != nil && withCookies {
		cookieStr := cookies.CookiesToString(c.cookies)
		if cookieStr != "" {
			headers.Set("cookie", cookieStr)
		}
		w, _ := c.cookies.GetViewports()
		headers.Set("viewport-width", w)
		headers.Set("x-asbd-id", "129477")
	}
	return headers
}

func (c *Client) addFacebookHeaders(h *http.Header) {
	if c.configs != nil {
		if c.configs.LsdToken != "" {
			h.Set("x-fb-lsd", c.configs.LsdToken)
		}
	}
}

func (c *Client) addInstagramHeaders(h *http.Header) {
	if c.configs != nil {
		csrfToken := c.cookies.GetValue("csrftoken")
		mid := c.cookies.GetValue("mid")
		if csrfToken != "" {
			h.Set("x-csrftoken", csrfToken)
		}

		if mid != "" {
			h.Set("x-mid", mid)
		}

		if c.configs.browserConfigTable != nil {
			instaCookies, ok := c.cookies.(*cookies.InstagramCookies)
			if ok {
				if instaCookies.IgWWWClaim == "" {
					h.Set("x-ig-www-claim", "0")
				} else {
					h.Set("x-ig-www-claim", instaCookies.IgWWWClaim)
				}
			} else {
				h.Set("x-ig-www-claim", "0")
			}
		}
		h.Set("x-ig-app-id", c.configs.browserConfigTable.CurrentUserInitialData.AppID)
	}
}

func (c *Client) findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func (c *Client) sendLoginRequest(form url.Values, loginUrl string) (*http.Response, []byte, error) {
	h := c.buildLoginHeaders()
	loginPayload := []byte(form.Encode())

	resp, respBody, err := c.MakeRequest(loginUrl, "POST", h, loginPayload, types.FORM)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send login request: %v", err)
	}

	return resp, respBody, nil
}

func (c *Client) buildLoginHeaders() http.Header {
	h := c.buildHeaders(true)
	if c.platform.IsMessenger() {
		h = c.addLoginFacebookHeaders(h)
	} else {
		h = c.addLoginInstagramHeaders(h)
	}
	h.Set("origin", c.getEndpoint("base_url"))
	h.Set("referer", c.getEndpoint("login_page"))

	return h
}

func (c *Client) addLoginFacebookHeaders(h http.Header) http.Header {
	h.Set("sec-fetch-dest", "document")
	h.Set("sec-fetch-mode", "navigate")
	h.Set("sec-fetch-site", "same-origin") // header is required
	h.Set("sec-fetch-user", "?1")
	h.Set("upgrade-insecure-requests", "1")
	return h
}

func (c *Client) addLoginInstagramHeaders(h http.Header) http.Header {
	h.Set("x-instagram-ajax", strconv.Itoa(int(c.configs.browserConfigTable.SiteData.ServerRevision)))
	h.Set("sec-fetch-dest", "empty")
	h.Set("sec-fetch-mode", "cors")
	h.Set("sec-fetch-site", "same-origin") // header is required
	h.Set("x-requested-with", "XMLHttpRequest")
	return h
}
