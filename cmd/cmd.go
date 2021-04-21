package cmd

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gonejack/go-epub"
)

type MHTMLToEpub struct {
	DefaultCover []byte

	Cover   string
	Title   string
	Author  string
	Verbose bool

	book *epub.Epub
}

func (h *MHTMLToEpub) Run(mhts []string, output string) (err error) {
	if len(mhts) == 0 {
		return errors.New("no mht files given")
	}

	h.book = epub.NewEpub(h.Title)
	{
		h.setAuthor()
		h.setDesc()
		err = h.setCover()
		if err != nil {
			return
		}
	}

	for _, mht := range mhts {
		err = h.processMHT(mht)
		if err != nil {
			err = fmt.Errorf("parse %s failed: %s", mht, err)
			return
		}
	}

	err = h.book.Write(output)
	if err != nil {
		return fmt.Errorf("cannot write output epub: %s", err)
	}

	return
}

func (h *MHTMLToEpub) setAuthor() {
	h.book.SetAuthor(h.Author)
}
func (h *MHTMLToEpub) setDesc() {
	h.book.SetDescription(fmt.Sprintf("Epub generated at %s with github.com/gonejack/mhtml-to-epub", time.Now().Format("2006-01-02")))
}
func (h *MHTMLToEpub) setCover() (err error) {
	if h.Cover == "" {
		temp, err := os.CreateTemp("", "textbundle-to-epub")
		if err != nil {
			return fmt.Errorf("cannot create tempfile: %s", err)
		}
		_, err = temp.Write(h.DefaultCover)
		if err != nil {
			return fmt.Errorf("cannot write tempfile: %s", err)
		}
		_ = temp.Close()

		h.Cover = temp.Name()
	}

	fmime, err := mimetype.DetectFile(h.Cover)
	if err != nil {
		return fmt.Errorf("cannot detect cover mime type %s", err)
	}
	coverRef, err := h.book.AddImage(h.Cover, "epub-cover"+fmime.Extension())
	if err != nil {
		return fmt.Errorf("cannot add cover %s", err)
	}
	h.book.SetCover(coverRef, "")

	return
}

func (h *MHTMLToEpub) processMHT(path string) (err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}

	tp := textproto.NewReader(bufio.NewReader(&trimReader{rd: file}))

	// Parse the main headers
	headers, err := tp.ReadMIMEHeader()
	if err != nil {
		return
	}

	body := tp.R
	ps, err := parseMIMEParts(headers, body)
	if err != nil {
		return
	}

	var parts = make(map[string]*part)
	var html *part
	for _, p := range ps {
		if ct := p.header.Get("Content-Type"); ct == "" {
			return ErrMissingContentType
		}
		ct, _, err := mime.ParseMediaType(p.header.Get("Content-Type"))
		if err != nil {
			return err
		}

		if html == nil && ct == "text/html" {
			html = p
			continue
		}

		ref := p.header.Get("Content-Location")
		if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
			parts[ref] = p
		}
	}

	if html == nil {
		return errors.New("html not found")
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html.body))
	if err != nil {
		return
	}

	doc = h.cleanDoc(doc)
	doc.Find("img").Each(func(i int, img *goquery.Selection) {
		h.changImgRef(img, parts)
	})

	var internalCSS string
	doc.Find(`link[type="text/css"]`).Each(func(i int, link *goquery.Selection) {
		if internalCSS == "" {
			internalCSS = h.changeCSSRef(link, parts)
		}
	})

	title := doc.Find("title").Text()
	content, err := doc.Find("body").Html()
	if err != nil {
		return
	}

	_, err = h.book.AddSection(content, title, "", internalCSS)

	return
}
func (h *MHTMLToEpub) changImgRef(img *goquery.Selection, parts map[string]*part) {
	img.RemoveAttr("loading")
	img.RemoveAttr("srcset")

	src, _ := img.Attr("src")

	part, exist := parts[src]
	if !exist {
		return
	}

	if part.tempfile == "" {
		fp, err := os.CreateTemp("", "html2epub*")
		if err != nil {
			log.Printf("cannot create temp file for %s: %s", src, err)
			return
		}
		_, err = fp.Write(part.body)
		if err != nil {
			log.Printf("cannot write temp file for %s: %s", src, err)
			return
		}
		_ = fp.Close()

		part.tempfile = fp.Name()
	}

	// check mime
	fmime, err := mimetype.DetectFile(part.tempfile)
	if err != nil {
		log.Printf("cannot detect image mime of %s: %s", src, err)
		return
	}
	if !strings.HasPrefix(fmime.String(), "image") {
		log.Printf("mime of %s is %s instead of images", src, fmime.String())
		return
	}

	// add image
	internalName := md5str(src) + filepath.Ext(src)
	if !strings.HasSuffix(internalName, fmime.Extension()) {
		internalName += fmime.Extension()
	}

	internalRef, err := h.book.AddImage(part.tempfile, internalName)
	if err != nil {
		log.Printf("cannot add image %s", err)
		return
	}

	img.SetAttr("src", internalRef)
}
func (h *MHTMLToEpub) changeCSSRef(link *goquery.Selection, parts map[string]*part) (internalRef string) {
	src, _ := link.Attr("href")

	part, exist := parts[src]
	if !exist {
		return
	}

	if part.tempfile == "" {
		fp, err := os.CreateTemp("", "html2epub*")
		if err != nil {
			log.Printf("cannot create temp file for %s: %s", src, err)
			return
		}
		_, err = fp.Write(part.body)
		if err != nil {
			log.Printf("cannot write temp file for %s: %s", src, err)
			return
		}
		_ = fp.Close()

		part.tempfile = fp.Name()
	}

	// add image
	internalName := md5str(src) + ".css"
	internalRef, err := h.book.AddCSS(part.tempfile, internalName)
	if err != nil {
		log.Printf("cannot add image %s", err)
		return
	}

	return
}
func (h *MHTMLToEpub) cleanDoc(doc *goquery.Document) *goquery.Document {
	return doc
}

func md5str(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

// part is a copyable representation of a multipart.Part
type part struct {
	header   textproto.MIMEHeader
	body     []byte
	tempfile string
}

// trimReader is a custom io.Reader that will trim any leading
// whitespace, as this can cause email imports to fail.
type trimReader struct {
	rd      io.Reader
	trimmed bool
}

// Read trims off any unicode whitespace from the originating reader
func (tr *trimReader) Read(buf []byte) (int, error) {
	n, err := tr.rd.Read(buf)
	if err != nil {
		return n, err
	}
	if !tr.trimmed {
		t := bytes.TrimLeftFunc(buf[:n], unicode.IsSpace)
		tr.trimmed = true
		n = copy(buf, t)
	}
	return n, err
}
