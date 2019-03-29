package pprof

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/google/pprof/driver"
	"github.com/sbueringer/pprof-exporter/tmp/go/src/strconv"
	"os"
)

func NewFlagSet(params map[string]string, args ...string) driver.FlagSet {
	return flagSet{
		FlagSet: flag.NewFlagSet("", flag.ContinueOnError),
		params:  params,
		args:    args,
	}
}

type flagSet struct {
	*flag.FlagSet
	params map[string]string
	args   []string
}

func (f flagSet) StringList(name string, def string, usage string) *[]*string {
	return &[]*string{}
}

func (f flagSet) ExtraUsage() string {
	return "extraUsage"
}

func (f flagSet) Parse(usage func()) []string {
	return f.args
}

func (f flagSet) String(name, def, usage string) *string {
	value := f.params[name]
	return &value
}

func (f flagSet) Bool(name string, def bool, usage string) *bool {
	value, ok := f.params[name]
	if !ok {
		return &ok
	}
	boolVal, err := strconv.ParseBool(value)
	if err != nil {
		panic(err)
	}
	return &boolVal
}

type UI struct {
	r *bufio.Reader
}

func (ui *UI) ReadLine(prompt string) (string, error) {
	os.Stdout.WriteString(prompt)
	return ui.r.ReadString('\n')
}

func (ui *UI) Print(args ...interface{}) {
	fmt.Fprint(os.Stderr, args)
}

func (ui *UI) PrintErr(args ...interface{}) {
	fmt.Fprint(os.Stderr, args)
}

func (ui *UI) IsTerminal() bool {
	return false
}

func (ui *UI) WantBrowser() bool {
	return false
}

func (ui *UI) SetAutoComplete(func(string) string) {
}
