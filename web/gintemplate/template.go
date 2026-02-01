/*
 * Portions Copyright 2018 Foolin. All rights reserved.
 * Licensed under the MIT License.
 *
 * Portions Copyright 2019 lostvip.
 * Licensed under the Apache License, Version 2.0.
 *
 * Use of this source code is governed by a dual license:
 * - The original MIT license for Foolin's gin-template project
 * - The Apache License 2.0 for modifications and enhancements
 *
 * See the LICENSE file for details.
 */

/*
Golang template for gin framework, Use golang html/template syntax,
Easy and simple to use for gin framework, See https://github.com/foolin/gin-template
for more information.
*/
package gintemplate

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lostvip-com/lv_framework/lv_conf"
	gocache "github.com/patrickmn/go-cache"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
)

var (
	htmlContentType   = []string{"text/html; charset=utf-8"}
	templateEngineKey = "github.com/foolin/gin-template/templateEngine"
)

type TemplateEngine struct {
	config      TemplateConfig
	tplCache    *gocache.Cache
	tplMutex    sync.RWMutex
	fileHandler FileHandler
}

type TemplateRender struct {
	Engine *TemplateEngine
	Name   string
	Data   interface{}
}

type TemplateConfig struct {
	Root      string           //view root
	Extension string           //template extension
	Master    string           //template master
	Partials  []string         //template partial, such as head, foot
	Funcs     template.FuncMap //template functions
	Delims    Delims           //delimeters
	CacheTTL  time.Duration    //cache TTL for template content (0 = no cache)
}

type Delims struct {
	Left  string
	Right string
}

type FileHandler func(config TemplateConfig, tplFile string) (content string, err error)

func New(config TemplateConfig) *TemplateEngine {
	return &TemplateEngine{
		config:      config,
		tplCache:    gocache.New(config.CacheTTL, time.Minute),
		tplMutex:    sync.RWMutex{},
		fileHandler: DefaultFileHandler(),
	}
}

func Default() *TemplateEngine {
	// Get template cache TTL from config, default to 0 (no cache)
	// Set to positive duration like "1h" to enable caching
	cacheTTL := time.Duration(0)
	if ttlStr := lv_conf.Config().GetValueStr("application.template.cache-ttl"); ttlStr != "" {
		if duration, err := time.ParseDuration(ttlStr); err == nil {
			cacheTTL = duration
		}
	}
	DefaultConfig := TemplateConfig{
		Root:      "views",
		Extension: ".html",
		Master:    "layouts/master",
		Partials:  []string{},
		Funcs:     make(template.FuncMap),
		Delims:    Delims{Left: "{{", Right: "}}"},
		CacheTTL:  cacheTTL,
	}
	return New(DefaultConfig)
}

// You should use helper func `Middleware()` to set the supplied
// TemplateEngine and make `HTML()` work validly.
func HTML(ctx *gin.Context, code int, name string, data interface{}) {
	if val, ok := ctx.Get(templateEngineKey); ok {
		if e, ok := val.(*TemplateEngine); ok {
			e.HTML(ctx, code, name, data)
			return
		}
	}
	ctx.HTML(code, name, data)
}

// New gin middleware for func `gintemplate.HTML()`
func NewMiddleware(config TemplateConfig) gin.HandlerFunc {
	return Middleware(New(config))
}

func Middleware(e *TemplateEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(templateEngineKey, e)
	}
}

func (e *TemplateEngine) Instance(name string, data interface{}) render.Render {
	return TemplateRender{
		Engine: e,
		Name:   name,
		Data:   data,
	}
}

func (e *TemplateEngine) HTML(ctx *gin.Context, code int, name string, data interface{}) {
	instance := e.Instance(name, data)
	ctx.Render(code, instance)
}

func (e *TemplateEngine) executeRender(out io.Writer, name string, data interface{}) error {
	useMaster := true
	if filepath.Ext(name) == e.config.Extension {
		useMaster = false
		name = strings.TrimSuffix(name, e.config.Extension)

	}
	return e.executeTemplate(out, name, data, useMaster)
}

func (e *TemplateEngine) executeTemplate(out io.Writer, name string, data interface{}, useMaster bool) error {

	var err error
	allFuncs := make(template.FuncMap, 0)
	allFuncs["include"] = func(layout string) (template.HTML, error) {
		buf := new(bytes.Buffer)
		err := e.executeTemplate(buf, layout, data, false)
		return template.HTML(buf.String()), err
	}
	// Get the plugin collection
	for k, v := range e.config.Funcs {
		allFuncs[k] = v
	}

	// Try to get from cache first
	e.tplMutex.RLock()
	cachedTpl, found := e.tplCache.Get(name)
	e.tplMutex.RUnlock()

	var tpl *template.Template
	if found {
		tpl = cachedTpl.(*template.Template)
	} else {
		e.tplMutex.Lock()
		// Double-check after acquiring write lock
		cachedTpl, found = e.tplCache.Get(name)
		if !found {
			tpl, err = e.LoadTpl(useMaster, name, allFuncs)
			if err != nil {
				e.tplMutex.Unlock()
				return err
			}
			if tpl == nil {
				e.tplMutex.Unlock()
				return fmt.Errorf("failed to load template: %s, error: %v", name, err)
			}
			// Cache the parsed template (will expire automatically based on CacheTTL)
			e.tplCache.Set(name, tpl, e.config.CacheTTL)
		} else {
			tpl = cachedTpl.(*template.Template)
		}
		e.tplMutex.Unlock()
	}

	exeName := name
	if useMaster && e.config.Master != "" {
		exeName = e.config.Master
	}

	// Display the content to the screen
	err = tpl.Funcs(allFuncs).ExecuteTemplate(out, exeName, data)
	if err != nil {
		return fmt.Errorf("TemplateEngine execute template error: %v", err)
	}

	return nil
}

func (e *TemplateEngine) LoadTpl(useMaster bool, name string, allFuncs template.FuncMap) (*template.Template, error) {
	tplList := make([]string, 0)
	if useMaster {
		//render()
		if e.config.Master != "" {
			tplList = append(tplList, e.config.Master)
		}
	}
	tplList = append(tplList, name)
	tplList = append(tplList, e.config.Partials...)

	// Loop through each template and test the full path
	tpl := template.New(name).Funcs(allFuncs).Delims(e.config.Delims.Left, e.config.Delims.Right)
	for _, v := range tplList {
		var data string
		data, err := e.fileHandler(e.config, v)
		if err != nil {
			return nil, err
		}
		var tmpl *template.Template
		if v == name {
			tmpl = tpl
		} else {
			tmpl = tpl.New(v)
		}
		_, err = tmpl.Parse(data)
		if err != nil {
			return nil, fmt.Errorf("TemplateEngine render parser name:%v, error: %v", v, err)
		}
	}
	return tpl, nil
}

func (e *TemplateEngine) SetFileHandler(handle FileHandler) {
	if handle == nil {
		panic("FileHandler can't set nil!")
	}
	e.fileHandler = handle
}

// ClearCache clears all cached templates
// Use this when templates are updated and you need to refresh the cache
func (e *TemplateEngine) ClearCache() error {
	e.tplMutex.Lock()
	e.tplCache.Flush()
	e.tplMutex.Unlock()
	return nil
}

// ClearTemplateCache clears a specific template from cache
func (e *TemplateEngine) ClearTemplateCache(templateName string) error {
	e.tplMutex.Lock()
	e.tplCache.Delete(templateName)
	e.tplMutex.Unlock()
	return nil
}

func (r TemplateRender) Render(w http.ResponseWriter) error {
	return r.Engine.executeRender(w, r.Name, r.Data)
}

func (r TemplateRender) WriteContentType(w http.ResponseWriter) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = htmlContentType
	}
}

func DefaultFileHandler() FileHandler {
	return func(config TemplateConfig, tplFile string) (content string, err error) {
		// Get the absolute path of the root template
		templatePath := filepath.Join(config.Root, tplFile+config.Extension)
		path, err := filepath.Abs(templatePath)
		if err != nil {
			return "", fmt.Errorf("TemplateEngine path:%v error: %v", path, err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("TemplateEngine render read name:%v, path:%v, error: %v", tplFile, path, err)
		}
		content = string(data)
		return content, nil
	}
}
