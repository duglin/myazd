package main

import (
	"bytes"
	"fmt"
	"strings"
)

type Form struct {
	Parent *Form
	Type   string // Section, Prop, Array(section)

	Title string
	Items []*Form // If Group  - if not empty then it's a prop
	Value string  // If prop

	Space bool // Blank line befor this Item

	// Used by diff, add, sub
	prevOne string
}

func NewForm() *Form {
	return &Form{
		Parent: nil,
		Type:   "Section",
	}
}

func (f *Form) GetProp(title string) string {
	for _, item := range f.Items {
		if item.Title == title {
			return item.Value
		}
	}
	return ""
}

func (f *Form) AddItem(item *Form) *Form {
	item.Parent = f
	f.Items = append(f.Items, item)
	return f
}

func (f *Form) AddSection(title string, value string) *Form {
	newForm := NewForm()
	newForm.Parent = f
	newForm.Type = "Section"
	newForm.Title = title
	newForm.Value = value
	newForm.Space = (len(f.Items) > 0)
	f.AddItem(newForm)
	return newForm
}

func (f *Form) AddArray(title string, value string) *Form {
	newForm := NewForm()
	newForm.Parent = f
	newForm.Type = "Array"
	newForm.Title = title
	newForm.Value = value
	newForm.Space = (len(f.Items) > 0)
	f.AddItem(newForm)
	return newForm
}

func (f *Form) AddProp(name string, value string) *Form {
	prop := &Form{
		Parent: f,
		Type:   "Prop",
		Title:  name,
		Value:  value,
	}
	f.AddItem(prop)
	return prop
}

func (f *Form) ToString() string {
	if f == nil {
		return ""
	}
	return f.ToStringContext(&context{
		indent:      "",
		prevIsSpace: true,
	})
}

type context struct {
	indent        string
	prevIsSpace   bool
	propNameWidth int
	arrayIndex    int // 1-based, NOT 0
}

func (f *Form) Dump() {
	f.DumpIndent("")
}

func (f *Form) DumpIndent(indent string) {
	fmt.Printf("%s%s/%s", indent, f.Type, f.Title)
	if f.Value != "" {
		fmt.Printf(": %s", f.Value)
	}
	fmt.Print("\n")

	for _, item := range f.Items {
		item.DumpIndent(indent + "  ")
	}
}

func (f *Form) ToStringContext(ctx *context) string {
	return f.NewToStringContext(ctx, "")
}

func (f *Form) NewToStringContext(ctx *context, indent string) string {
	buf := &bytes.Buffer{}
	// fmt.Printf("Form: %s/%s indent: %q\n", f.Type, f.Title, indent)

	if f.Space {
		fmt.Fprint(buf, "\n")
	}

	doTitle := (f.Title != "" && f.Title[0] != '*')
	if doTitle {
		width := ""
		if ctx.propNameWidth > 0 && f.Type == "Prop" {
			width = fmt.Sprintf("-%d", ctx.propNameWidth)
		}
		fmt.Fprintf(buf, "%s%"+width+"s:", indent, f.Title)
		if f.Value != "" {
			fmt.Fprintf(buf, " %s", f.Value)
		}
		fmt.Fprint(buf, "\n")
		indent += "  "
		// indent = strings.ReplaceAll(indent, "-", " ")
	}

	saveWidth := ctx.propNameWidth
	if len(f.Items) > 0 {
		ctx.propNameWidth = 0
		for _, item := range f.Items {
			if item == nil {
				continue
			}
			if item.Type == "Prop" && len(item.Title) > ctx.propNameWidth {
				ctx.propNameWidth = len(item.Title)
			}
		}
	}
	if f.Type == "Array" {
		indent += "- "
	}
	for _, item := range f.Items {
		if item == nil {
			continue
		}
		buf.WriteString(item.NewToStringContext(ctx, indent))
		if item.Type == "Prop" && item.Parent != nil &&
			item.Parent.Type != "Array" {
			indent = strings.ReplaceAll(indent, "-", " ")
		}
	}
	ctx.propNameWidth = saveWidth

	return buf.String()
}

func (f *Form) TToStringContext(ctx *context) string {
	// fmt.Printf("%s: %s(%s,%s)\n", ctx.indent, f.Type, f.Title, f.Value)
	buf := &bytes.Buffer{}

	if f.Space { // && !ctx.prevIsSpace {
		fmt.Fprintf(buf, "\n")
		ctx.prevIsSpace = true
	}

	indent := ctx.indent
	if (ctx.arrayIndex > 0 && f.Type != "Prop") || (ctx.arrayIndex == 1 && f.Type == "Prop") {
		if l := len(indent); l > 1 {
			indent = indent[:l-2] + "-" + indent[l-1:]
		}
	}

	if f.Type == "Section" || f.Type == "Array" {
		if f.Title != "" && f.Title[0] != '*' {
			fmt.Fprintf(buf, "%s%s:%s\n", indent, f.Title, f.Value)
			ctx.indent += "  "
		}

		ctx.propNameWidth = 0
		for _, item := range f.Items {
			if item.Type == "Prop" && len(item.Title) > ctx.propNameWidth {
				ctx.propNameWidth = len(item.Title)
			}
		}

		for i, item := range f.Items {
			if f.Type == "Array" || (f.Parent != nil && f.Parent.Type == "Array" && f.Type == "Section" && (f.Title == "" || f.Title[0] == '*')) {
				ctx.arrayIndex = i + 1
				if f.Type == "Array" {
					ctx.indent += "  "
				}
			} else {
				ctx.arrayIndex = 0
			}
			fmt.Fprintf(buf, "%s", item.ToStringContext(ctx))
			if f.Type == "Array" {
				ctx.indent = ctx.indent[:len(ctx.indent)-2]
			}
		}

		if f.Title != "" && f.Title[0] != '*' {
			ctx.indent = ctx.indent[:len(ctx.indent)-2]
		}
	} else if f.Type == "Prop" {
		if f.Space {
			if !ctx.prevIsSpace {
				fmt.Fprintf(buf, "\n")
				ctx.prevIsSpace = true
			}
			ctx.propNameWidth = 0
		}
		width := ""
		if ctx.propNameWidth > 0 {
			width = fmt.Sprintf("-%d", ctx.propNameWidth)
		}
		fmt.Fprintf(buf, "%s%"+width+"s: %s\n", indent, f.Title, f.Value)
		ctx.prevIsSpace = false
	} else {
		panic("Bad type: " + f.Type)
	}

	return buf.String()
}

func CalcItemsWidth(f *Form, index int) int {
	width := 0
	for index < len(f.Items) {
		if f.Items[index].Title == "" || len(f.Items[index].Items) != 0 {
			break
		}
		if len(f.Items[index].Title) > width {
			width = len(f.Items[index].Title)
		}
	}

	return width
}

type diffContext struct {
	title       string
	srcName     string
	tgtName     string
	shownLegend bool
	lastTitle   string
	sync        bool
	all         bool
}

func (dc *diffContext) showLegend() {
	if dc.shownLegend {
		return
	}
	// fmt.Printf("< %s\n", dc.srcName)
	// fmt.Printf("> %s\n", dc.tgtName)
	fmt.Printf("%s\n", dc.title)
	dc.shownLegend = true
}

func (dc *diffContext) showTitle(title string) {
	if title != dc.lastTitle {
		fmt.Printf("\n## %s\n", title)
		dc.lastTitle = title
	} else {
		fmt.Printf("\n")
	}
}

func (f *Form) Sub(subF *Form) *Form {
	if f.Type != subF.Type {
		panic("Not same type: " + f.Type + "/" + subF.Type)
	}
	resF := f.CloneNoItems()

	if f.Title != subF.Title || f.Value != subF.Value {
		return f.Clone() // Not the same Item, don't touch it
	}
	if f.Type == "Prop" { // Prop match, so remove it
		return nil
	}

	// Remove any items in src's Items that appear in sub's Items
	for i, fItem := range f.Items {
		for _, subItem := range subF.Items {
			if subItem == nil || fItem.Type != subItem.Type {
				continue // no item, or not of the right Type
			}
			if fItem.Title != subItem.Title {
				continue
			}
			if fItem.Type == "Section" || fItem.Type == "Array" {
				if fItem.Value != fItem.Value {
					continue
				}
			}
			// Found a match, so try to 'sub' it
			fItem = fItem.Sub(subItem)
			break
		}
		if fItem == f.Items[i] { // didn't find a match so clone it
			fItem = fItem.Clone()
		}
		if fItem != nil { // didn't remove it, so clone it
			if i > 0 {
				fItem.prevOne = f.Items[i-1].Title + ":" + f.Items[i-1].Value
			}
			resF.AddItem(fItem)
		}
	}
	if len(resF.Items) == 0 { // && resF.Value == ""
		return nil
	}

	return resF
}

func (f *Form) Patch(addF *Form) {
	if f.Type != addF.Type {
		panic("Not same type: " + f.Type + "/" + addF.Type)
	}

	if f.Title != addF.Title {
		return
	}

	if f.Type == "Prop" {
		f.Value = f.Value
		return
	}

	for _, aItem := range addF.Items {
		found := false
		addAfter := -1
		for i, fItem := range f.Items {
			search := (f.Items[i].Value + ":" + f.Items[i].Value)
			if aItem.prevOne == search {
				addAfter = i
			}

			if fItem.Type != aItem.Type || fItem.Title != aItem.Title {
				continue
			}
			if fItem.Type == "Prop" {
				fItem.Value = aItem.Value
				found = true
				break
			}
			// Section or Array
			if fItem.Value == aItem.Value {
				fItem.Patch(aItem) // recurse on this item
				found = true
			}
		}
		if !found {
			if addAfter == -1 {
				f.Items = append(f.Items, aItem.Clone())
			} else {
				f.Items = append(f.Items[:addAfter],
					append([]*Form{aItem.Clone()}, f.Items[addAfter:]...)...)
			}
		}
	}
}

func (f *Form) Clone() *Form {
	newF := f.CloneNoItems()
	for _, item := range f.Items {
		newF.AddItem(item.Clone())
	}
	return newF
}

func (f *Form) CloneNoItems() *Form {
	newF := &Form{
		Parent:  nil, // Set by caller
		Type:    f.Type,
		Title:   f.Title,
		Items:   nil,
		Value:   f.Value,
		Space:   f.Space,
		prevOne: f.prevOne,
	}
	return newF
}

func (f *Form) MiniToString() string {
	if f.Type == "Prop" {
		return fmt.Sprintf("{%s:%s}", f.Title, f.Value)
	}
	str := fmt.Sprintf("{%s:%s", f.Title, f.Value)
	wrapper := "{}"
	if f.Type == "Array" {
		wrapper = "[]"
	}
	str = fmt.Sprintf("%s%v", str, wrapper[0])
	for i, item := range f.Items {
		if i > 0 {
			str += ","
		}
		str += item.MiniToString()
	}
	str = fmt.Sprintf("%s%v}", str, wrapper[1])
	return str
}

func (srcForm *Form) Diff(tgtForm *Form, dc *diffContext) {
	// fmt.Printf("Diffing: %s/%s\n", srcForm.Type, srcForm.Title)
	// Section, Prop, Array

	if srcForm.Type != tgtForm.Type {
		panic("Not same type: " + srcForm.Type + "/" + tgtForm.Type)
	}

	if srcForm.Type == "Prop" {
		if srcForm.Value != tgtForm.Value {
			dc.showLegend()
			dc.showTitle(srcForm.GenContext())
			// fmt.Printf("srcForm: %#v   title:%s\n", srcForm, title)
			// fmt.Printf("parent: %s/%s\n", srcForm.Parent.Type, srcForm.Parent.Title)
			fmt.Printf("< %s: %s\n", srcForm.Title, srcForm.Value)
			fmt.Printf("> %s: %s\n", tgtForm.Title, tgtForm.Value)

			if dc.sync {
				res := byte('a')
				if !dc.all {
					res = Prompt(fmt.Sprintf("a)ccept r)eject changes "))
				}
				if res == 'a' {
					srcForm.Value = tgtForm.Value
				}
			}
		}
	} else if srcForm.Type == "Section" || srcForm.Type == "Array" {
		srcIndexes := make([]int, len(srcForm.Items))
		for i, srcItem := range srcForm.Items {
			srcIndexes[i] = findItem(tgtForm.Items, srcItem)
		}

		tgtIndexes := make([]int, len(tgtForm.Items))
		for i, tgtItem := range tgtForm.Items {
			tgtIndexes[i] = findItem(srcForm.Items, tgtItem)
		}

		// fmt.Printf("Diff: %s/%s\n", srcForm.Type, srcForm.Title)
		// fmt.Printf("  srcInd: %v\n", srcIndexes)
		// fmt.Printf("  tgtInd: %v\n", tgtIndexes)

		srcI, tgtI := 0, 0
		srcItems, tgtItems := srcForm.Items[:], tgtForm.Items[:]

		newItems := []*Form{}
		for srcI < len(srcIndexes) || tgtI < len(tgtIndexes) {
			// fmt.Printf("srcI: %d  tgtI: %d\n", srcI, tgtI)
			// Show all src items (at front of list) not in tgt
			if srcI < len(srcIndexes) && srcIndexes[srcI] == -1 {
				item := srcItems[srcI]
				dc.showLegend()
				dc.showTitle(item.GenContext())
				item.Space = false
				fmt.Printf("%s",
					item.NewToStringContext(&context{
						indent:      "<   ",
						prevIsSpace: true,
					}, "< "))
				if dc.sync {
					res := byte('a')
					if !dc.all {
						res = Prompt(fmt.Sprintf("a)ccept r)eject existing "))
					}
					if res == 'a' {
						// fmt.Printf("Item: %s\n", item.ToString())
						newItems = append(newItems, item)
					}
				}
				srcI++
				continue
			}
			if srcI < len(srcIndexes) && srcItems[srcI] == nil {
				srcI++
				continue
			}

			// Show all tgt items (at front of list) not in src
			if tgtI < len(tgtIndexes) && tgtIndexes[tgtI] == -1 {
				item := tgtItems[tgtI]
				dc.showLegend()
				dc.showTitle(item.GenContext())
				item.Space = false
				fmt.Printf("%s",
					item.NewToStringContext(&context{
						indent:      ">   ",
						prevIsSpace: true,
					}, "> "))
				if dc.sync {
					res := byte('a')
					if !dc.all {
						res = Prompt(fmt.Sprintf("a)ccept R)eject addition "))
					}
					if res == 'a' {
						newItems = append(newItems, item)
					}
				}
				tgtI++
				continue
			}
			if tgtI < len(tgtIndexes) && tgtItems[tgtI] == nil {
				tgtI++
				continue
			}

			// Assume src is the preferred order, so go find it in tgt
			// and diff it, then remove it from tgt's list
			inTgt := srcIndexes[srcI]

			if srcItems[srcI].Title != tgtItems[inTgt].Title {
				panic(fmt.Sprintf("Diff name mismatch: %q vs %q",
					srcItems[srcI].Title, tgtItems[inTgt].Title))
			}

			srcItems[srcI].Diff(tgtItems[inTgt], dc)
			newItems = append(newItems, srcItems[srcI])

			srcItems[srcI] = nil  // technically not needed
			tgtItems[inTgt] = nil // avoids processing this one again later

			srcI++
			if tgtI == inTgt {
				tgtI++
			}
		}
		if dc.sync {
			srcForm.Items = newItems[:]
		}
	}
}

func findItem(items []*Form, searchItem *Form) int {
	for i, item := range items {
		itemStr := item.Title
		searchStr := searchItem.Title

		if item.Type != "Prop" {
			itemStr += ":" + item.Value
			searchStr += ":" + searchItem.Value
		}

		if itemStr == searchStr {
			return i
		}
	}
	return -1
}

func (form *Form) GenContext() string {
	res := ""
	// fmt.Printf("Gen: %#v\n", form)
	for p := form.Parent; p != nil; p = p.Parent {
		if p.Title != "" { // && p.Title[0] != '*' {
			title := p.Title
			if title[0] == '*' {
				title = title[1:]
			}
			if res == "" {
				res = title
			} else {
				res = title + "/" + res
			}
		}
	}
	if res == "" {
		res = "(Resource)"
	}
	return res
}
