// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package discover

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"time"

	"golang.org/x/tools/go/packages"
)

// Layout describes the queues and schedules found in user code.
type Layout struct {
	Queues    []Queue
	Schedules []Schedule
}

type Queue struct {
	Name string
}

type Schedule struct {
	Name  string
	Every time.Duration
}

// Scan walks user code to discover registered queues and schedules.
// Reflection is used only at build time; runtime assets remain reflection-free.
func Scan(dir string) (Layout, error) {
	cfg := &packages.Config{
		Mode: packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedFiles | packages.NeedDeps,
		Dir:  dir,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return Layout{}, err
	}

	queues := map[string]struct{}{}
	schedules := map[string]Schedule{}

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				switch sel.Sel.Name {
				case "RegisterQueueHandler":
					if len(call.Args) < 1 {
						return true
					}
					if name := stringValue(pkg, call.Args[0]); name != "" {
						queues[name] = struct{}{}
					}
				case "RegisterScheduleHandler":
					if len(call.Args) < 2 {
						return true
					}
					name := stringValue(pkg, call.Args[0])
					if name == "" {
						return true
					}
					dur := durationValue(pkg, call.Args[1])
					schedules[name] = Schedule{Name: name, Every: dur}
				}

				return true
			})
		}
	}

	var layout Layout
	for name := range queues {
		layout.Queues = append(layout.Queues, Queue{Name: name})
	}
	for _, sched := range schedules {
		layout.Schedules = append(layout.Schedules, sched)
	}
	return layout, nil
}

func parseBasicString(raw string) (string, error) {
	if len(raw) < 2 {
		return "", fmt.Errorf("invalid string literal: %s", raw)
	}
	return raw[1 : len(raw)-1], nil
}

func stringValue(pkg *packages.Package, expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.BasicLit:
		if v.Kind != token.STRING {
			return ""
		}
		val, err := parseBasicString(v.Value)
		if err != nil {
			return ""
		}
		return val
	case *ast.Ident:
		if pkg.TypesInfo == nil {
			return ""
		}
		tv, ok := pkg.TypesInfo.Types[expr]
		if !ok || tv.Value == nil {
			return ""
		}
		if tv.Value.Kind() != constant.String {
			return ""
		}
		return constant.StringVal(tv.Value)
	default:
		return ""
	}
}

func durationValue(pkg *packages.Package, expr ast.Expr) time.Duration {
	if pkg.TypesInfo == nil {
		return 0
	}
	tv, ok := pkg.TypesInfo.Types[expr]
	if !ok || tv.Value == nil {
		return 0
	}
	c := constant.ToInt(tv.Value)
	if c.Kind() == constant.Int {
		if v, ok := constant.Int64Val(c); ok {
			return time.Duration(v)
		}
	}
	return 0
}
