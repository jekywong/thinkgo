// Copyright 2016 henrylee2cn.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.
package core

import (
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"
)

type Think struct {
	Echo *Echo
	// 模块列表
	Modules map[string]*Module
	// 插件列表
	// Addons *Addons
	// 模板引擎
	*Template
	// 配置信息
	Config
	// 框架信息
	Author  string
	Version string
}

// 重要配置，涉及项目架构，请勿修改
const (
	// 模块应用目录名
	APP_PACKAGE = "application"
	// 视图文件目录名
	VIEW_PACKAGE = "view"
	// 公共目录
	COMMON_PACKAGE = "common"
	// 资源文件目录名
	PUBLIC_PACKAGE = "__public__"
	// 上传根目录名
	UPLOADS_PACKAGE = "uploads"
)

// 全局运行实例
var ThinkGo = func() *Think {
	t := &Think{
		// 业务数据
		Echo:    New(),
		Modules: Modules,
		Config:  getConfig(),
		// 框架信息
		Author:  AUTHOR,
		Version: VERSION,
	}

	log := t.Echo.Logger()
	log.SetPrefix("TG")
	t.Echo.Use(Recover(), Logger())
	t.Echo.Blackfile(".html")
	t.Echo.SetLogLevel(t.Config.LogLevel)
	t.Echo.SetDebug(t.Config.Debug)
	t.htmlPrepare()
	t.dirServe()
	t.Hook()
	// t.Echo.SetBinder(b)
	// t.Echo.SetHTTPErrorHandler(HTTPErrorHandler)
	// t.Echo.SetLogOutput(w io.Writer)
	// t.Echo.SetHTTPErrorHandler(h HTTPErrorHandler)
	return t
}()

func (this *Think) Run() {
	this.Echo.Run(fmt.Sprintf("%s:%d", this.Config.HttpAddr, this.Config.HttpPort))
}

func (this *Think) dirServe() {
	this.Echo.ServeFile("/favicon.ico", "deploy/favicon/favicon.ico")
	this.Echo.ServeDir("/uploads", UPLOADS_PACKAGE)
	this.Echo.ServeDir("/common", APP_PACKAGE+"/"+COMMON_PACKAGE+"/"+VIEW_PACKAGE+"/"+PUBLIC_PACKAGE)

	var re = regexp.MustCompile(APP_PACKAGE + "(/[^/]+)/" + VIEW_PACKAGE + "(/[^/]+)/" + PUBLIC_PACKAGE)
	for _, p := range WalkRelDirs(APP_PACKAGE, "/"+PUBLIC_PACKAGE) {
		a := re.FindStringSubmatch(p)
		if len(a) == 3 {
			// public/[模块]/[主题]/
			this.Echo.ServeDir(path.Join("/public", a[1], a[2]), p)
		}
	}
	if this.Echo.Debug() {
		for k, v := range this.Template.Map() {
			this.Echo.logger.Notice("	%-25s --> %-25s", k, v)
		}
	}
}

func (this *Think) htmlPrepare() {
	var t = NewRender()
	t.Delims(this.Config.TplLeft, this.Config.TplRight)
	t.SetBasepath(APP_PACKAGE)
	t.SetSuffix(this.Config.TplSuffix)
	t.SetDebug(this.Config.Debug)

	var (
		paths []string
		re    = regexp.MustCompile(t.basepath + "(/[^/]+)/" + VIEW_PACKAGE + "(/[^/]+)(/[^/]+)(/[^/]+)" + t.suffix)
		re2   = regexp.MustCompile(t.basepath + "/" + COMMON_PACKAGE + "/" + VIEW_PACKAGE + "(/[^/]+)" + t.suffix)
	)

	for _, f := range WalkRelFiles(t.basepath, t.suffix) {
		a := re.FindStringSubmatch(f)
		if len(a) < 5 {
			b := re2.FindStringSubmatch(f)
			if len(b) == 2 {
				t.pathmap["/common"+b[1]] = f
				paths = append(paths, f)
			}
			continue
		}
		r := a[1] + a[2] + a[3] + a[4]
		t.pathmap[r] = f
		paths = append(paths, f)
	}
	if !t.debug {
		t.Template.ParseFiles(paths...)
	}

	t.Template.Delims(t.delims[0], t.delims[1])

	this.Template = t
	this.Echo.SetRenderer(t)
}

func (this *Think) Hook() {
	this.Echo.Hook(func(w http.ResponseWriter, r *http.Request) {
		fs := this.Echo.fileSystem.path
		if fs != "" && strings.HasPrefix(r.URL.Path, fs) {
			return
		}
		p := strings.Trim(r.URL.Path, "/")
		// 补全默认模块
		switch p {
		case "favicon.ico":
			return
		case "":
			p = this.Config.DefaultModule
		default:
			idx := strings.Index(p, "/")
			var m string
			if idx == -1 {
				m = p
			} else {
				m = p[:idx]
			}
			if _, ok := this.Modules[m]; !ok {
				switch m {
				case "common", "public", "uploads":
					return
				default:
					p = this.Config.DefaultModule + "/" + p
				}
			}
		}
		// 补全名为index的控制器或操作
		num := 2 - strings.Count(p, "/")
		if num > 0 {
			p += strings.Repeat("/index", num)
		}

		// 转换url模式
		if r.URL.RawQuery == "" {
			ps := strings.Split(p, "/")
			num := len(ps) - 3
			if num <= 0 {
				goto end
			}
			for ; num > 0; num-- {
				r.URL.RawQuery += fmt.Sprintf("&%v=%v", num-1, ps[2+num])
			}
		end:
			r.URL.Path = path.Join("/", ps[0], ps[1], ps[2])
		} else {
			r.URL.Path = path.Join("/", p)
		}
	})
}

func (this *Think) Group(prefix string, m ...Middleware) *Group {
	return this.Echo.Group(prefix, m...)
}
