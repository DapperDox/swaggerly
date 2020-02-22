/*
Copyright (C) 2016-2017 dapperdox.com

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.

*/
package asset

// This package scans a default template directory *and* an override template directory,
// building an "go generate" compliant Assets structure. Templates in under the override
// directory replace of suppliment those in the default directory.
// This allows default themes to be provided, and then changed on a per-use basis by
// dropping files in the override directory.

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/DapperDox/dapperdox/config"
	"github.com/DapperDox/dapperdox/logger"
	"github.com/shurcooL/github_flavored_markdown"
)

var _bindata = map[string][]byte{}
var _metadata = map[string]map[string]string{}
var guideReplacer *strings.Replacer
var gfmReplace []*gfmReplacer

var sectionSplitRegex = regexp.MustCompile("\\[\\[[\\w\\-\\/]+\\]\\]")
var gfmMapSplit = regexp.MustCompile(":")

// ---------------------------------------------------------------------------
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if a, ok := _bindata[cannonicalName]; ok {
		return a, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// ---------------------------------------------------------------------------
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// ---------------------------------------------------------------------------
func MetaData(filename string, name string) string {
	if md, ok := _metadata[filename]; ok {
		if val, ok := md[strings.ToLower(name)]; ok {
			return val
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
func MetaDataFileList() []string {
	files := make([]string, len(_metadata))
	ix := 0
	for key := range _metadata {
		files[ix] = key
		ix++
	}
	return files
}

// ---------------------------------------------------------------------------
func Compile(dir string, prefix string) {

	cfg, _ := config.Get()

	// Build a replacer to search/replace Document URLs in the documents.
	if guideReplacer == nil {
		var replacements []string

		// Configure the replacer with key=value pairs
		for i := range cfg.DocumentRewriteURL {

			slice := strings.Split(cfg.DocumentRewriteURL[i], "=")

			if len(slice) != 2 {
				panic("Invalid DocumentWriteUrl - does not contain an = delimited from=to pair")
			}
			replacements = append(replacements, slice...)
		}
		guideReplacer = strings.NewReplacer(replacements...)
	}

	dir, err := filepath.Abs(dir)
	if err != nil {
		logger.Errorf(nil, "Error forming absolute path: %s", err)
	}

	logger.Debugf(nil, "- Scanning directory %s", dir)

	dir = filepath.ToSlash(dir)

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		path = filepath.Clean(filepath.ToSlash(path))

		if info == nil {
			return nil
		}
		if info.IsDir() {
			// Skip hidden directories TODO this should be applied to files also.
			_, node := filepath.Split(path)
			if node[0] == '.' {
				return filepath.SkipDir
			}
			return nil
		}

		ext := ""
		if strings.Index(path, ".") != -1 {
			ext = filepath.Ext(path)
		}

		buf, err := ioutil.ReadFile(path)
		if err != nil {
			panic(err)
		}

		relative, err := filepath.Rel(dir, path)
		if err != nil {
			panic(err)
		}

		var meta map[string]string

		switch ext {
		// The file may be in GFM, so convert to HTML and process any embedded metadata
		case ".md":
			// Chop off the extension
			mdname := strings.TrimSuffix(relative, ext)

			buf, meta = ProcessMetadata(buf)

			// This resource may be metadata tagged as a page section overlay..
			if overlay, ok := meta["overlay"]; ok && strings.ToLower(overlay) == "true" {

				// Chop markdown into sections
				sections, headings := splitOnSection(string(buf))

				if sections == nil {
					logger.Errorf(nil, "  * Error no sections defined in overlay file %s\n", relative)
					os.Exit(1)
				}

				for i, heading := range headings {
					buf = ProcessMarkdown([]byte(sections[i]))

					relative = filepath.Join(mdname, heading, "overlay.tmpl")
					storeTemplate(prefix, relative, guideReplacer.Replace(string(buf)), meta)
				}
			} else {
				buf = ProcessMarkdown(buf) // Convert markdown into HTML

				relative = mdname + ".tmpl"
				storeTemplate(prefix, relative, guideReplacer.Replace(string(buf)), meta)
			}
		case ".tmpl":
			buf, meta = ProcessMetadata(buf)
			storeTemplate(prefix, relative, guideReplacer.Replace(string(buf)), meta)

		case ".html":
			logger.Errorf(nil, "  * Error - Refusing to process .html files. Expects HTML template fragments with .tmpl extension. File %s\n", relative)
			os.Exit(1)

		default:
			storeTemplate(prefix, relative, guideReplacer.Replace(string(buf)), meta)
		}

		return nil
	})
}

// ---------------------------------------------------------------------------

func storeTemplate(prefix string, name string, template string, meta map[string]string) {

	newname := filepath.ToSlash(filepath.Join(prefix, name))

	if _, ok := _bindata[newname]; !ok {
		logger.Debugf(nil, "  + Import %s", newname)
		// Store the template, doing and search/replaces on the way
		_bindata[newname] = []byte(template)
		if len(meta) > 0 {
			logger.Tracef(nil, "    + Adding metadata")
			_metadata[newname] = meta
		}
	}
}

// ---------------------------------------------------------------------------
// Returns rendered markdown
func ProcessMarkdown(doc []byte) []byte {

	html := github_flavored_markdown.Markdown([]byte(doc))
	// Apply any HTML substitutions
	for _, rep := range gfmReplace {
		html = rep.Regexp.ReplaceAll(html, rep.Replace)
	}
	return html
}

// ---------------------------------------------------------------------------
// Strips and processed metadata from markdown document
func ProcessMetadata(doc []byte) ([]byte, map[string]string) {

	// Inspect the markdown src doc to see if it contains metadata
	reader := bytes.NewReader(doc)
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)

	var newdoc string
	metaData := make(map[string]string)

	for scanner.Scan() {
		line := scanner.Text()
		splitLine := strings.Split(line, ":")

		trimmed := strings.TrimSpace(splitLine[0])
		if (len(splitLine) < 2) || (!unicode.IsLetter(rune(trimmed[0]))) { // Have we reached a non KEY: line? If so, we're done with the metadata.
			if len(line) > 0 { // If the line is not empty, keep the contents
				newdoc = newdoc + line + "\n"
			}
			// Gather up all remainging lines
			for scanner.Scan() {
				// TODO Make this more efficient!
				newdoc = newdoc + scanner.Text() + "\n"
			}
			break
		}

		// Else, deal with meta-data
		metaValue := ""
		if len(splitLine) > 1 {
			metaValue = strings.TrimSpace(splitLine[1])
		}

		metaKey := strings.ToLower(splitLine[0])
		metaData[metaKey] = metaValue
	}

	return []byte(newdoc), metaData
}

// ---------------------------------------------------------------------------

func splitOnSection(text string) ([]string, []string) {

	indexes := sectionSplitRegex.FindAllStringIndex(text, -1)

	if indexes == nil {
		return nil, nil
	}

	last := 0
	sections := make([]string, len(indexes))
	headings := make([]string, len(indexes))

	for i, element := range indexes {

		if i > 0 {
			sections[i-1] = text[last:element[0]]
		}

		headings[i] = text[element[0]+2 : element[1]-2] // +/- 2 removes the leading/trailing [[ ]]

		last = element[1]
	}
	sections[len(indexes)-1] = text[last:]

	return sections, headings
}

// ---------------------------------------------------------------------------

func CompileGFMMap() {

	var mapfile string

	cfg, _ := config.Get()

	if len(cfg.AssetsDir) != 0 {
		mapfile = filepath.Join(cfg.AssetsDir, "gfm.map")
		logger.Tracef(nil, "Looking in assets dir for %s\n", mapfile)
		if _, err := os.Stat(mapfile); os.IsNotExist(err) {
			mapfile = ""
		}
	}
	if len(mapfile) == 0 && len(cfg.ThemeDir) != 0 {
		mapfile = filepath.Join(cfg.ThemeDir, cfg.Theme, "gfm.map")
		logger.Tracef(nil, "Looking in theme dir for %s\n", mapfile)
		if _, err := os.Stat(mapfile); os.IsNotExist(err) {
			mapfile = ""
		}
	}
	if len(mapfile) == 0 {
		mapfile = filepath.Join(cfg.DefaultAssetsDir, "themes", cfg.Theme, "gfm.map")
		logger.Tracef(nil, "Looking in default theme dir for %s\n", mapfile)
		if _, err := os.Stat(mapfile); os.IsNotExist(err) {
			mapfile = ""
		}
	}
	if len(mapfile) == 0 {
		mapfile = filepath.Join(cfg.DefaultAssetsDir, "/themes/default/gfm.map")
		logger.Tracef(nil, "Looking in default theme for %s\n", mapfile)
		if _, err := os.Stat(mapfile); os.IsNotExist(err) {
			mapfile = ""
		}
	}

	if len(mapfile) == 0 {
		logger.Tracef(nil, "No GFM HTML mapfile found\n")
		return
	}
	logger.Tracef(nil, "Processing GFM HTML mapfile: %s\n", mapfile)
	file, err := os.Open(mapfile)

	if err != nil {
		logger.Errorf(nil, "Error: %s", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		rep := &gfmReplacer{}
		if rep.Parse(line) != nil {
			logger.Tracef(nil, "GFM replace %s with %s\n", rep.Regexp, rep.Replace)
			gfmReplace = append(gfmReplace, rep)
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Errorf(nil, "Error: %s", err)
	}
}

// ---------------------------------------------------------------------------
type gfmReplacer struct {
	Regexp  *regexp.Regexp
	Replace []byte
}

// ---------------------------------------------------------------------------
func (g *gfmReplacer) Parse(line string) *string {
	indexes := gfmMapSplit.FindStringIndex(line)
	if indexes == nil {
		return nil
	}
	g.Regexp = regexp.MustCompile(line[0 : indexes[1]-1])
	g.Replace = []byte(line[indexes[1]:])

	return &line
}

// ---------------------------------------------------------------------------
// end
