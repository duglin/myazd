package main

import (
	"fmt"
)

type Form struct {
	Parent *Form
	Type   string // Section, Prop, Array(section)

	Title string
	// Exactly one of these 2
	Items []*Form // If Group  - if not empty then it's a prop
	Value string  // If prop

	Space    bool // Blank line befor this Item
	Together bool // All or nothing

	Indent string
	Align  bool // Align values in this section (colons), not span spaces
}

func NewForm() *Form {
	return &Form{
		Parent: nil,
		Type:   "Section",
		Align:  true,
	}
}

func (f *Form) AddSection(title string) *Form {
	newForm := NewForm()
	newForm.Parent = f
	newForm.Type = "Section"
	newForm.Title = title
	f.Items = append(f.Items, newForm)
	return newForm
}

func (f *Form) AddArray(title string) *Form {
	newForm := NewForm()
	newForm.Parent = f
	newForm.Type = "Array"
	newForm.Title = title
	f.Items = append(f.Items, newForm)
	return newForm
}

func (f *Form) AddProp(name string, value string) *Form {
	prop := &Form{
		Parent: f,
		Type:   "Prop",
		Title:  name,
		Value:  value,
	}
	f.Items = append(f.Items, prop)
	return prop
}

func (f *Form) Print() {
	f.PrintContext(&context{
		indent:      "",
		prevIsSpace: true,
	})
}

type context struct {
	indent        string
	prevIsSpace   bool
	propNameWidth int
}

func (f *Form) PrintContext(ctx *context) {
	if f.Space && !ctx.prevIsSpace {
		fmt.Printf("\n")
		ctx.prevIsSpace = true
	}

	if f.Type == "Section" || f.Type == "Array" {
		saveCtx := *ctx
		if f.Title != "" {
			fmt.Printf("%s%s:\n", ctx.indent, f.Title)
			ctx.indent += "  "
			ctx.prevIsSpace = false
			saveCtx.prevIsSpace = false
		}

		if f.Align {
			ctx.propNameWidth = 0
			for _, item := range f.Items {
				if item.Type == "Prop" && len(item.Title) > ctx.propNameWidth {
					ctx.propNameWidth = len(item.Title)
				}
			}
		}

		for i, item := range f.Items {
			if f.Type == "Array" {
				if i == 0 {
					ctx.indent += "- "
				} else {
					ctx.indent += "  "
				}
			}
			item.PrintContext(ctx)
			if f.Type == "Array" {
				ctx.indent = ctx.indent[:len(ctx.indent)-2]
			}
		}

		if f.Title != "" {
			ctx.indent = ctx.indent[:len(ctx.indent)-2]
		}
	} else if f.Type == "Prop" {
		if f.Space {
			if !ctx.prevIsSpace {
				fmt.Printf("\n")
				ctx.prevIsSpace = true
			}
			ctx.propNameWidth = 0
		}
		width := ""
		if ctx.propNameWidth > 0 {
			width = fmt.Sprintf("-%d", ctx.propNameWidth)
		}
		fmt.Printf("%s%"+width+"s: %s\n", ctx.indent, f.Title, f.Value)
		ctx.prevIsSpace = false
	} else if f.Type == "Array" {
	} else {
		panic("Bad type: " + f.Type)
	}
}

func CalcItemsWidth(items []*Form, index int) int {
	width := 0
	for index < len(items) {
		if items[index].Title == "" || len(items[index].Items) != 0 {
			break
		}
		if len(items[index].Title) > width {
			width = len(items[index].Title)
		}
	}

	return width
}
