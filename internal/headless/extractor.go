package headless

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/playwright-community/playwright-go"
)

// ExtractAllImageURLs extracts image URLs from the page covering img, picture, background-image, CSS, meta tags, svg, video poster, etc.
func ExtractAllImageURLs(page playwright.Page, baseURL string) ([]string, error) {
	err := page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateDomcontentloaded,
	})

	_, _ = page.Evaluate(`async () => {
		await new Promise(resolve => {
			let total = 0;
			const h = Math.max(document.documentElement.clientHeight, window.innerHeight || 0);
			function s(i) {
				if (i>100) { resolve(); return; }
				window.scrollBy(0, h);
				total += h;
				setTimeout(()=>s(i+1), 150);
			}
			s(0);
		});
	}`, nil)

	// JS: collect image URLs from many sources
	js := `() => {
		const urls = new Set();
		const add = (u) => { if (!u) return; try { u = u.trim(); if (u && u !== 'none') urls.add(u); } catch(e){} };

		// helper: parse srcset string to candidates
		const addSrcset = (ss) => {
			if (!ss) return;
			const parts = ss.split(',');
			for (const p of parts) {
				const candidate = p.trim().split(/\s+/)[0];
				if (candidate) add(candidate);
			}
		};

		// 1) <img> tags
		document.querySelectorAll('img').forEach(img => {
			add(img.getAttribute('src'));
			add(img.getAttribute('data-src'));
			add(img.getAttribute('data-lazy'));
			add(img.getAttribute('data-srcset'));
			addSrcset(img.getAttribute('srcset'));
		});

		// 2) <picture> -> <source>
		document.querySelectorAll('picture source').forEach(s => {
			add(s.getAttribute('src'));
			addSrcset(s.getAttribute('srcset'));
		});

		// 3) <video poster>
		document.querySelectorAll('video[poster]').forEach(v => add(v.getAttribute('poster')));

		// 4) <svg> <image xlink:href / href>
		document.querySelectorAll('svg image').forEach(im => {
			add(im.getAttribute('href') || im.getAttributeNS('http://www.w3.org/1999/xlink','href'));
		});

		// 5) meta og:image and link rel=image_src
		const og = document.querySelector('meta[property="og:image"], meta[name="og:image"]');
		if (og) add(og.getAttribute('content'));
		const linkImg = document.querySelector('link[rel="image_src"]');
		if (linkImg) add(linkImg.getAttribute('href'));

		// 6) inline style background-image and computed style (including ::before and ::after)
		const reUrl = /url\(\s*['"]?(.*?)['"]?\s*\)/g;
		document.querySelectorAll('*').forEach(el => {
			// inline style
			const inline = el.getAttribute('style');
			if (inline) {
				let m;
				while ((m = reUrl.exec(inline)) !== null) add(m[1]);
			}
			// computed style
			try {
				const cs = window.getComputedStyle(el);
				if (cs && cs.getPropertyValue('background-image')) {
					let m;
					while ((m = reUrl.exec(cs.getPropertyValue('background-image'))) !== null) add(m[1]);
				}
				// pseudo elements
				['::before','::after'].forEach(p => {
					const pcs = window.getComputedStyle(el, p);
					if (pcs) {
						let m;
						while ((m = reUrl.exec(pcs.getPropertyValue('background-image'))) !== null) add(m[1]);
					}
				});
			} catch(e) {}
		});

		// 7) CSS rules in stylesheets -> find url(...) patterns
		for (const sheet of Array.from(document.styleSheets)) {
			let rules;
			try { rules = sheet.cssRules || sheet.rules; } catch(e) { continue; }
			if (!rules) continue;
			for (const r of Array.from(rules)) {
				const text = r.cssText || '';
				let m;
				while ((m = reUrl.exec(text)) !== null) add(m[1]);
			}
		}

		// normalize: remove data URIs if you don't want them, but we'll keep them
		return Array.from(urls);
	}`

	// Evaluate and get back []interface{}
	res, err := page.Evaluate(js, nil)
	if err != nil {
		return nil, fmt.Errorf("evaluate js error: %w", err)
	}

	// res is usually []interface{}
	rawList, ok := res.([]interface{})
	if !ok {
		// fallback: maybe it's already []string
		if sarr, ok2 := res.([]string); ok2 {
			rawList = make([]interface{}, len(sarr))
			for i, v := range sarr {
				rawList[i] = v
			}
		} else {
			return nil, fmt.Errorf("unexpected result type from evaluate: %T", res)
		}
	}

	// convert to string, parse srcset items, resolve, dedupe
	seen := map[string]bool{}
	var out []string
	srcsetRe := regexp.MustCompile(`\s*,\s*`)
	urlPartRe := regexp.MustCompile(`^([^ \t]+)`)                // first token in srcset candidate
	noiseRe := regexp.MustCompile(`(?i)(favicon\.|sprite|logo)`) // <-- added

	for _, it := range rawList {
		if it == nil {
			continue
		}
		raw := strings.TrimSpace(fmt.Sprintf("%v", it))
		if raw == "" {
			continue
		}

		// Skip unwanted schemes
		if strings.HasPrefix(raw, "data:") || strings.HasPrefix(raw, "blob:") || strings.HasPrefix(raw, "javascript:") {
			continue
		}

		// Skip common noise assets
		if noiseRe.MatchString(raw) {
			continue
		}

		// Handle srcset-style lists
		if strings.Contains(raw, ",") && strings.Contains(raw, " ") {
			parts := srcsetRe.Split(raw, -1)
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				m := urlPartRe.FindStringSubmatch(p)
				if len(m) > 1 {
					u := resolveURL(baseURL, m[1])
					if !seen[u] {
						seen[u] = true
						out = append(out, u)
					}
				}
			}
			continue
		}

		// Normal single URL
		u := resolveURL(baseURL, raw)
		if !seen[u] {
			seen[u] = true
			out = append(out, u)
		}
	}

	return out, nil
}

// resolveURL resolves relative URLs against base; keep protocol-relative and data URIs
func resolveURL(base, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// keep data: and javascript: untouched
	if strings.HasPrefix(raw, "data:") || strings.HasPrefix(raw, "javascript:") {
		return raw
	}
	// protocol-relative
	if strings.HasPrefix(raw, "//") {
		u, err := url.Parse("https:" + raw) // assume https
		if err == nil {
			return u.String()
		}
	}
	// try absolute
	u, err := url.Parse(raw)
	if err == nil && u.IsAbs() {
		return u.String()
	}
	// else resolve with base
	baseu, err := url.Parse(base)
	if err != nil {
		// fallback: return raw
		return raw
	}
	res := baseu.ResolveReference(u)
	return res.String()
}
