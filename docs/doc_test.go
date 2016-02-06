// Copyright 2016 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Test documentation validity.
package docs

import (
	"bytes"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/russross/blackfriday"
	"github.com/stretchr/testify/require"
)

// Directory to start crawling for docs from.
const rootDir = "../client/README.md" // FIXME

var (
	// Files and directory patterns to ignore.
	ignore = []string{
		"Godeps",
		".[^.]*", // Ignore hidden files.
	}

	// Expectations for where these links should point to.
	linkLineExpectations = map[string]string{
		"ContainerInfo struct":             "type ContainerInfo struct {",
		"MachineInfo struct in the source": "type MachineInfo struct {",
	}
)

type Framework struct {
	*testing.T
}

func TestDocs(t *testing.T) {
	f := &Framework{T: t}
	require.NoError(t, filepath.Walk(rootDir, f.walk))
}

type DocTest struct {
	*Framework

	// The path to the file being tested.
	path string
}

func (d *DocTest) Logf(format string, args ...interface{}) {}

// Test the link for soundness.
func (d *DocTest) CheckLink(text, title, href string) {
	u, err := url.Parse(href)
	if err != nil {
		d.Errorf("[%s] Error parsing URL %q: %v", d.path, href, err)
		return
	}

	if u.IsAbs() {
		if u.Host == "github.com" {
			// Links to cAdvisor should be relative.
			if strings.HasPrefix(u.Path, "/google/cadvisor") {
				d.Errorf("[%s] %q should be relative", d.path, href)
			}
		} else {
			// Ignore non-github links.
			d.Logf("[%s] not github: %q", d.path, u.Host)
			return
		}
	}

	lineRegexp := regexp.MustCompile(`^(?:.*&)?L([0-9]+)(?:&.*)?$`)
	match := lineRegexp.FindStringSubmatch(u.Fragment)
	if len(match) == 2 {
		if title == "" {
			d.Errorf("[%s] github line links should have a title identifying the content: %q", d.path, href)
			return
		}
		expect, ok := linkLineExpectations[title]
		if !ok {
			d.Errorf("[%s] no expectation set for %q (%q)", d.path, title, href)
			return
		}

		linum, err := strconv.Atoi(match[1])
		if err != nil {
			d.Errorf("[%s] %v", d.path, err)
		}

		_, _ = expect, linum // FIXME
	}

	d.Logf("[%s] no match: %q", d.path, u.Fragment)
}

//
// The following is the test framework for checking the markdown.
//

// Markdown renderer options.
const extensions = 0 |
	blackfriday.EXTENSION_FENCED_CODE |
	blackfriday.EXTENSION_AUTOLINK |
	blackfriday.EXTENSION_NO_EMPTY_LINE_BEFORE_BLOCK

func (f *Framework) checkFile(path string) error {
	input, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	f.Logf("Checking %q", path)
	d := &DocTest{f, path}
	blackfriday.Markdown(input, d, extensions)
	return nil
}

func (f *Framework) walk(path string, info os.FileInfo, err error) error {
	if err != nil {
		return nil // Ignore file errors.
	}
	for _, pattern := range ignore {
		matched, err := filepath.Match(pattern, info.Name())
		if err != nil {
			return err
		}
		if matched {
			if info.IsDir() {
				return filepath.SkipDir // Ignore directory
			} else {
				return nil // Ignore file
			}
		}
	}
	if info.IsDir() {
		return nil
	}
	// Only check markdown files.
	matched, err := filepath.Match("*.md", info.Name())
	if err != nil {
		return err
	}
	if !matched {
		return nil
	}
	return f.checkFile(path)
}

// DocTest implements blackfriday.Renderer to process markdown.
var _ blackfriday.Renderer = &DocTest{}

func (d *DocTest) BlockCode(out *bytes.Buffer, text []byte, lang string) {
	// d.Logf("[%s] found code: %s", d.path, string(text))
}
func (d *DocTest) BlockQuote(out *bytes.Buffer, text []byte) {}
func (d *DocTest) BlockHtml(out *bytes.Buffer, text []byte)  {}
func (d *DocTest) Header(out *bytes.Buffer, text func() bool, level int, id string) {
	d.advance(out, text)
}
func (d *DocTest) HRule(out *bytes.Buffer)                                               {}
func (d *DocTest) ListItem(out *bytes.Buffer, text []byte, flags int)                    {}
func (d *DocTest) Paragraph(out *bytes.Buffer, text func() bool)                         { d.advance(out, text) }
func (d *DocTest) Table(out *bytes.Buffer, header []byte, body []byte, columnData []int) {}
func (d *DocTest) TableRow(out *bytes.Buffer, text []byte)                               {}
func (d *DocTest) TableHeaderCell(out *bytes.Buffer, text []byte, flags int)             {}
func (d *DocTest) TableCell(out *bytes.Buffer, text []byte, flags int)                   {}
func (d *DocTest) Footnotes(out *bytes.Buffer, text func() bool)                         { d.advance(out, text) }
func (d *DocTest) FootnoteItem(out *bytes.Buffer, name, text []byte, flags int)          {}
func (d *DocTest) TitleBlock(out *bytes.Buffer, text []byte)                             {}
func (d *DocTest) CodeSpan(out *bytes.Buffer, text []byte)                               {}
func (d *DocTest) DoubleEmphasis(out *bytes.Buffer, text []byte)                         {}
func (d *DocTest) Emphasis(out *bytes.Buffer, text []byte)                               {}
func (d *DocTest) Image(out *bytes.Buffer, link []byte, title []byte, alt []byte)        {}
func (d *DocTest) LineBreak(out *bytes.Buffer)                                           {}
func (d *DocTest) RawHtmlTag(out *bytes.Buffer, tag []byte)                              {}
func (d *DocTest) TripleEmphasis(out *bytes.Buffer, text []byte)                         {}
func (d *DocTest) StrikeThrough(out *bytes.Buffer, text []byte)                          {}
func (d *DocTest) FootnoteRef(out *bytes.Buffer, ref []byte, id int)                     {}
func (d *DocTest) Entity(out *bytes.Buffer, entity []byte)                               {}
func (d *DocTest) NormalText(out *bytes.Buffer, text []byte) {
	d.Logf("[%s] found text: %s", d.path, string(text))
	out.Write(text)
}
func (d *DocTest) DocumentHeader(out *bytes.Buffer) {}
func (d *DocTest) DocumentFooter(out *bytes.Buffer) {}
func (d *DocTest) AutoLink(out *bytes.Buffer, link []byte, kind int) {
	d.Logf("[%s] found autolink: %s", d.path, string(link))
}
func (d *DocTest) Link(out *bytes.Buffer, link []byte, title []byte, content []byte) {
	d.Logf("[%s] found link: %q", d.path, string(link))
	d.CheckLink(string(content), string(title), string(link))
}
func (d *DocTest) List(out *bytes.Buffer, text func() bool, flags int) { d.advance(out, text) }
func (d *DocTest) GetFlags() int {
	return 0
}
func (d *DocTest) advance(out *bytes.Buffer, text func() bool) {
	// Workaround for github.com/russross/blackfriday/issues/189
	marker := out.Len()
	if !text() {
		out.Truncate(marker)
	}
}
